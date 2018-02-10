package main

import (
  "context"
  "flag"
  "time"
  "fmt"
  "net"
  "strings"
  "errors"
  "strconv"

  "github.com/envoyproxy/go-control-plane/api"

  "github.com/envoyproxy/go-control-plane/pkg/cache"

  gcp "github.com/envoyproxy/go-control-plane/pkg/server"

  "github.com/gregdurham/consul-envoy-xds/agent"

  xdsConfig "github.com/gregdurham/consul-envoy-xds/config"
  
  "github.com/gregdurham/consul-envoy-xds/lib"

  "github.com/golang/glog"
  "google.golang.org/grpc"

  "github.com/cskr/pubsub"

  "net/http"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "github.com/grpc-ecosystem/go-grpc-prometheus"
)

// Hasher is a single cache key hash.
type Hasher struct {
}

func (h Hasher) Hash(node *api.Node) (cache.Key, error) {
  return cache.Key(node.GetId()), nil
}

type Service struct {
	name, resourceType string
}

type ServiceConfig struct {
  name, resourceType, resourceName string
}

type Resources map[string][]interface{}

var (
  httpPort     uint
  xdsPort      uint
  interval     time.Duration
  ads          bool
  consulHost   string
  consulPort   uint
  configPath   string
)

var ResultQueue = make(chan interface{}, 1000)
var ErrorQueue = make(chan error, 1000)

func init() {
  flag.UintVar(&xdsPort, "xds", 18000, "xDS server port")
  flag.UintVar(&httpPort, "http", 18080, "Http server port")
  flag.DurationVar(&interval, "interval", 10*time.Second, "Interval between cache refresh")
  flag.BoolVar(&ads, "ads", false, "Use ADS instead of separate xDS services")
  flag.StringVar(&consulHost, "consulHost", "127.0.0.1", "consul hostname/ip")
  flag.UintVar(&consulPort, "consulPort", 8500, "consul port")
  flag.StringVar(&configPath, "configPath", "envoy/", "consul kv path to configuration root")
}

func main() {
  flag.Parse()

  ctx := context.Background()

  config := cache.NewSimpleCache(Hasher{}, nil)
  consulUrl := fmt.Sprintf("%s:%d", consulHost, consulPort)
  a := agent.NewAgent(consulUrl, "", "dc1")

  //RunCacheUpdate(ctx, config, ads, interval, upstreamPort, listenPort)
  events := pubsub.New(1000)

	go RunCacheUpdate(ctx, config, consulUrl, configPath, a, events)

  server := gcp.NewServer(config)
  grpcServer := grpc.NewServer(
    grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
    grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
  )
  lis, err := net.Listen("tcp", fmt.Sprintf(":%d", xdsPort))
  if err != nil {
    glog.Fatalf("failed to listen: %v", err)
  }
  api.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
  api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
  api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
  api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
  api.RegisterListenerDiscoveryServiceServer(grpcServer, server)
  glog.Infof("xDS server listening on %d", xdsPort)
  go func() {
    if err = grpcServer.Serve(lis); err != nil {
      glog.Error(err)
    }
  }()
  grpc_prometheus.Register(grpcServer)
  httpServer := &http.Server{Addr: fmt.Sprintf(":%d", httpPort)}
  glog.Infof("http server listening on %d", httpPort)
  go func() {
    http.Handle("/metrics", promhttp.Handler())
    if err = httpServer.ListenAndServe(); err != nil {
      glog.Error(err)
    }
  }()
  if err := httpServer.Shutdown(ctx); err != nil {
    glog.Error(err)
  }
  <-ctx.Done()
  grpcServer.GracefulStop()


}

func RunCacheUpdate(ctx context.Context, 
  config cache.Cache,
  consulUrl string, 
  configPath string,
  a agent.ConsulAgent, 
  events *pubsub.PubSub) {

  serviceEndpoints := lib.NewEndpointIndex()

  watchers := make(map[string]lib.Watcher)
  updaters := make(map[string]lib.Updater)

  configWatcher := lib.NewWatcher("keyprefix", configPath, consulUrl, serviceEndpoints, events, ErrorQueue)
  configWatcher.WatchPlan()
  configWatcher.Start()
  subscriptionChannel := events.Sub(configPath)

  for {
    resourceMap, err := ConfigConsulKV(configPath, a)
    if err != nil {
      glog.V(10).Infof("Received the following error from the tag parser: %s", err)
      break
    }

    for service, resources := range resourceMap {
      if cacheUpdater, ok := updaters[service]; ok {
        cacheUpdater.RefreshConfig(resources)
      } else {
        updater := lib.NewUpdater(service, config, events, serviceEndpoints, ErrorQueue)
        updater.SetResources(resources)
        updater.Start()
        updaters[service] = updater
      } 
      for _, resource := range resources {
        if cluster, ok := resource.(xdsConfig.Cluster); ok {
          if cluster.GetName() != "local_service" {
            if _ ,ok = watchers[cluster.GetName()]; !ok {
              watch := lib.NewWatcher("service", cluster.GetName(), consulUrl, serviceEndpoints, events, ErrorQueue)
              watch.WatchPlan()
              watch.Start()
              watchers[cluster.GetName()] = watch
            }
          }
        }
      }
    }
    select {
    case <-ctx.Done():
      return
    case <- subscriptionChannel:
    }
  }
}

func ConfigConsulKV(configPath string, a agent.ConsulAgent) (Resources, error){
    serviceConfigs, _ := a.ListKeys(configPath)
    mapping := make(map[ServiceConfig]map[string]string)

    for _, servicePath := range serviceConfigs {
      if strings.HasSuffix(servicePath.Key, "/") {
        continue
      }

      svc := strings.TrimPrefix(servicePath.Key, configPath)
      /*
        0: service name
        1: type (cluster/listener)
        2: type name (cluster name/listener name)
        3: option name
      */
      
      splitServicePath := strings.Split(svc, "/")
      if len(splitServicePath) != 4 {
        continue
      }
      if _, ok := mapping[ServiceConfig{splitServicePath[0], splitServicePath[1], splitServicePath[2]}]; ok {
        mapping[ServiceConfig{splitServicePath[0], splitServicePath[1], splitServicePath[2]}][splitServicePath[3]] = string(servicePath.Value[:])
      } else {
        mapping[ServiceConfig{splitServicePath[0], splitServicePath[1], splitServicePath[2]}] = make(map[string]string)
        mapping[ServiceConfig{splitServicePath[0], splitServicePath[1], splitServicePath[2]}]["name"] = splitServicePath[2]
        mapping[ServiceConfig{splitServicePath[0], splitServicePath[1], splitServicePath[2]}][splitServicePath[3]] = string(servicePath.Value[:])
      }
    }
    serviceMap := make(Resources)
    for k, v := range mapping {
      resource, err := ParseConfigConsul(k.resourceType, v)
      if err != nil {
        return nil, err
      }
      serviceMap[k.name] = append(serviceMap[k.name], resource)
    }
    return serviceMap, nil
}

func ConfigConsulTags(a agent.ConsulAgent) (Resources, error){
  services, _ := a.CatalogServices()
  serviceMap := make(Resources)
  for service, tags := range services {
    for _, tag := range tags {
      resource, err := ParseConfig(tag)
      if err != nil {
        return nil, err
      }
      serviceMap[service] = append(serviceMap[service], resource)
    }
  }
  return serviceMap, nil
}


func ParseConfig(configString string) (interface{}, error) {
  var conf interface{}

  options := strings.Split(configString, ";")

  if (options[0] == "listener") {
    conf = xdsConfig.NewListener()
  } else if (options[0] == "cluster") {
    conf = xdsConfig.NewCluster()
  } else {
    return nil, errors.New("Not a config string")
  }

  for i := 1; i < len(options); i++ {
    option := strings.Split(options[i], "|")
    if len(option) != 2 {
      break
    } 
    key := strings.Title(option[0])
    value := option[1]

    fieldType, err := lib.GetFieldType(&conf, key)
    if err != nil {
      return nil, err
    }
    if fieldType == "int" {
      value, err := strconv.Atoi(value)
      if err != nil {
        return nil, errors.New("Could not convert string to int")
      }
      lib.SetIntField(&conf, key, int64(value))
    } else if fieldType == "string" {
      lib.SetStringField(&conf, key, value)
    } else if fieldType == "bool" {
      value, err := strconv.ParseBool(value)
      if err != nil {
        return nil, errors.New("Could not convert string to boolean")
      }
      lib.SetBoolField(&conf, key, value)
    } else if fieldType == "[]string" {
      items := strings.Split(value, ",")
      for _, item := range items {
        lib.AppendToSlice(&conf, key, item)
      }
    }else {
      return nil, errors.New("Value type not recoginized, must be one of string, int, or boolean")
    }
  }
  return conf, nil
}


func ParseConfigConsul(resourceType string, mapping map[string]string) (interface{}, error) {
  var conf interface{}

  if (resourceType == "listener") {
    conf = xdsConfig.NewListener()
  } else if (resourceType == "cluster") {
    conf = xdsConfig.NewCluster()
  } else {
    return nil, errors.New("Not a config string")
  }

  for k, v := range mapping {
    key := strings.Title(k)
    value := v

    fieldType, err := lib.GetFieldType(&conf, key)
    if err != nil {
      return nil, err
    }
    if fieldType == "int" {
      value, err := strconv.Atoi(value)
      if err != nil {
        return nil, errors.New("Could not convert string to int")
      }
      lib.SetIntField(&conf, key, int64(value))
    } else if fieldType == "string" {
      lib.SetStringField(&conf, key, value)
    } else if fieldType == "bool" {
      value, err := strconv.ParseBool(value)
      if err != nil {
        return nil, errors.New("Could not convert string to boolean")
      }
      lib.SetBoolField(&conf, key, value)
    } else if fieldType == "[]string" {
      items := strings.Split(value, ",")
      for _, item := range items {
        lib.AppendToSlice(&conf, key, item)
      }
    }else {
      return nil, errors.New("Value type not recoginized, must be one of string, int, or boolean")
    }
  }
  return conf, nil
}



