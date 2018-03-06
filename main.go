package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"strings"

	"sync"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/envoyproxy/go-control-plane/pkg/cache"

	gcp "github.com/envoyproxy/go-control-plane/pkg/server"

	"github.com/gregdurham/consul-envoy-service-mesh/agent"

	"github.com/gregdurham/consul-envoy-service-mesh/lib"

	"github.com/golang/glog"
	"google.golang.org/grpc"

	"github.com/cskr/pubsub"

	"net/http"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	consul_api "github.com/hashicorp/consul/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Hasher is a single cache key hash.
type Hasher struct {
}

func (h Hasher) ID(node *envoy_api_v2_core.Node) string {
	return node.GetId()
}

var (
	httpPort   uint
	xdsPort    uint
	consulHost string
	consulPort uint
	consulDC   string
	configPath string
)

var ResultQueue = make(chan interface{}, 1000)
var ErrorQueue = make(chan error, 1000)

func init() {
	flag.UintVar(&xdsPort, "xds", 18000, "xDS server port")
	flag.UintVar(&httpPort, "http", 18080, "Http server port")
	flag.StringVar(&consulHost, "consulHost", "127.0.0.1", "consul hostname/ip")
	flag.UintVar(&consulPort, "consulPort", 8500, "consul port")
	flag.StringVar(&configPath, "configPath", "envoy/", "consul kv path to configuration root")
	flag.StringVar(&consulDC, "consulDC", "dc1", "consul datacenter")
}

func main() {
	flag.Parse()

	ctx := context.Background()

	config := cache.NewSnapshotCache(true, Hasher{}, nil)

	consulUrl := fmt.Sprintf("%s:%d", consulHost, consulPort)
	a := agent.NewAgent(consulUrl, "", consulDC)

	events := pubsub.New(1000)

	go RunCacheUpdate(ctx, config, consulUrl, configPath, a, events)

	server := gcp.NewServer(cache.Cache(config), nil)
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", xdsPort))
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}
	envoy_service_discovery_v2.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	envoy_api_v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	envoy_api_v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	envoy_api_v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	envoy_api_v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)
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
	config cache.SnapshotCache,
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
			glog.Errorf("Invalid or malformed json: %s", err)
		} else {
			for service, resources := range resourceMap {
				if cacheUpdater, ok := updaters[service]; ok {
					cacheUpdater.RefreshConfig(resources)
				} else {
					updater := lib.NewUpdater(service, config, events, serviceEndpoints, ErrorQueue)
					updater.SetResources(resources)
					updater.Start()
					updaters[service] = updater
				}
				for _, cluster := range resources.Clusters {
					if cluster.Name != "local_service" {
						if _, ok := watchers[cluster.Name]; !ok {
							watch := lib.NewWatcher("service", cluster.Name, consulUrl, serviceEndpoints, events, ErrorQueue)
							watch.WatchPlan()
							watch.Start()
							watchers[cluster.Name] = watch
						}
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-subscriptionChannel:
		}
	}
}

func unmarshalConfig(mapping *lib.ConfigMapping, config *consul_api.KVPair) error {
	var v lib.Configs
	if err := json.Unmarshal(config.Value, &v); err != nil {

		return fmt.Errorf("Invalid JSON in %s", config.Key)
	}
	for _, config := range v {
		if cluster, ok := config.(*lib.ClusterConfig); ok {
			mapping.M.Lock()
			mapping.Cluster[cluster.Name] = *cluster
			mapping.M.Unlock()
		} else if listener, ok := config.(*lib.ListenerConfig); ok {
			mapping.M.Lock()
			mapping.Listerner[listener.Name] = *listener
			mapping.M.Unlock()
		} else if service, ok := config.(*lib.ServiceConfig); ok {
			mapping.M.Lock()
			mapping.Service[service.Name] = *service
			mapping.M.Unlock()
		}
	}
	return nil

}

func ConfigConsulKV(configPath string, a agent.ConsulAgent) (lib.Resources, error) {
	serviceConfigs, err := a.ListKeys(configPath)
	mapping := lib.NewConfigMapping()

	if err != nil {
		return nil, err
	}
	wg := sync.WaitGroup{}
	for _, servicePath := range serviceConfigs {
		if strings.HasSuffix(servicePath.Key, "/") {
			continue
		}
		wg.Add(1)
		go func(mapping *lib.ConfigMapping, config *consul_api.KVPair) {
			defer wg.Done()
			unmarshalConfig(mapping, config)
		}(mapping, servicePath)
	}
	wg.Wait()
	serviceMap := make(lib.Resources)

	for _, service := range mapping.Service {
		glog.V(10).Infof("Introspecting service %s", service.Name)
		resource := lib.NewResource()
		for _, listener := range service.Listeners {
			if listenerVal, ok := mapping.Listerner[listener]; ok {
				glog.V(10).Infof("Creating listener %s", listenerVal.Name)
				resource.Listeners = append(resource.Listeners, listenerVal)
				for _, cluster := range listenerVal.Clusters {
					if clusterVal, ok := mapping.Cluster[cluster]; ok {
						glog.V(10).Infof("Creating cluster %s", clusterVal.Name)
						resource.Clusters = append(resource.Clusters, clusterVal)
					} else {
						//cluster doesnt exist
						glog.Errorf("cluster %s does not exist", cluster)
					}
				}
			} else {
				// listener doesnt exist
				glog.Errorf("cluster %s does not exist", listener)
			}
		}
		serviceMap[service.Name] = *resource
	}
	return serviceMap, nil
}
