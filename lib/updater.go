package lib

import(
  "fmt"
  "github.com/cskr/pubsub"
  cpCache "github.com/envoyproxy/go-control-plane/pkg/cache"
  "github.com/gregdurham/consul-envoy-service-mesh/config"
  "github.com/gogo/protobuf/proto"
  "github.com/golang/glog"
)

type Updater struct {
  ID                string
  Listeners         []config.Listener
  Clusters          []config.Cluster
  Subscriptions     []string
  ServiceEndpoints  EndpointIndex
  Cache             cpCache.Cache
  Events            *pubsub.PubSub
  CmdChan           chan cmd
  ErrorChan         chan error
  SnapshotId        int
  grpcListeners     []proto.Message
  grpcClusters      []proto.Message
  grpcEndpoints     []proto.Message
  grpcRoutes        []proto.Message
}

type endpoint struct {
  address   string
  port      int
}

type cmd struct {
  command string
  data    []interface{}
}

func NewUpdater(id string, cache cpCache.Cache, events *pubsub.PubSub, serviceEndpoints EndpointIndex, errorChan chan error) Updater {
  updater := Updater{
    ID: id,
    Cache: cache,
    Events: events,
    ServiceEndpoints: serviceEndpoints,
    CmdChan: make(chan cmd),
    ErrorChan: errorChan,
    SnapshotId: 0,
  }

  return updater
}

func (u *Updater) SetResources(resources []interface{}) {
  listeners := make([]config.Listener, 0) 
  clusters := make([]config.Cluster, 0)
  subscriptions := []string{}
  for _, resource := range resources {
    if cluster, ok := resource.(config.Cluster); ok {
      clusters = append(clusters, cluster)
      subscriptions = append(subscriptions, cluster.GetName())
    } else if listener, ok := resource.(config.Listener); ok {
      listeners = append(listeners, listener)
    }
  }
  u.Listeners = listeners
  u.Clusters = clusters
  u.Subscriptions = subscriptions
  u.updateConfiguration()
}

func (u *Updater) Start() {
  glog.V(10).Infof("starting cache updater for %s", u.ID)
  SubChannel := u.Events.Sub(u.Subscriptions...)
  go func() {
    for {
      u.writeSnapshot()

      select {
        case cnc := <- u.CmdChan:
          if cnc.command == "quit" {

            glog.V(10).Infof("Received request to quit %s", u.ID)
            return
          } else if cnc.command == "refresh" {
            glog.V(10).Infof("Received request to refresh configuration %s", u.ID)
            u.SetResources(cnc.data)
          }
        case <- SubChannel:
          u.updateEndpoint()
      }
    }
  }()
}

func (u *Updater) updateEndpoint() {
  glog.V(10).Infof("Refreshing endpoints")
  endpoints := make([]proto.Message, 0)
  for _, cluster := range u.Clusters {
    endpoints = append(endpoints, u.createEndpoint(cluster))
  }
  
  u.grpcEndpoints = endpoints
}

func (u *Updater) writeSnapshot() {
  version := fmt.Sprintf("version%d", u.SnapshotId)
  snapshot := cpCache.NewSnapshot(version,
    u.grpcEndpoints,
    u.grpcClusters,
    u.grpcRoutes,
    u.grpcListeners)
  u.Cache.SetSnapshot(cpCache.Key(u.ID), snapshot)
  u.SnapshotId++
}

func (u *Updater) createListener(listener config.Listener) (proto.Message) {
  lstn := NewListener(listener, true)
  return lstn.Listener()
}

func (u *Updater) createCluster(cluster config.Cluster) (proto.Message) {
  cls := NewCluster(cluster, true)
  return cls.Cluster()
}

func (u *Updater) createEndpoint(cluster config.Cluster) (proto.Message) {
  if cluster.GetName() == "local_service" {
    svc := []*endpoint{{
      address: cluster.GetHost() ,
      port: cluster.GetPort(),
    }}
    ep := NewEndpoint(cluster.GetName(), svc)
    return ep.CLA()
  }
  if k, ok := u.ServiceEndpoints.GetEndpoints(cluster.GetName()); ok {
    ep := NewEndpoint(cluster.GetName(), k)
    return ep.CLA()
  }
  return nil
}

func (u *Updater) createRoute(listener string, clusters []config.Cluster) (proto.Message) {
  rt := NewRoute(listener, clusters)
  return rt.RouteCfg()
}

func (u *Updater) updateConfiguration() {
    listeners := make([]proto.Message, 0)
    clusters := make([]proto.Message, 0)
    endpoints := make([]proto.Message, 0)
    routes := make([]proto.Message, 0)

    resourceMapping := make(map[string][]config.Cluster, 0)

    for _, listener := range u.Listeners {
      listeners = append(listeners, u.createListener(listener))
    }

    for _, cluster := range u.Clusters {
      listener := cluster.GetListener()
      resourceMapping[listener] = append(resourceMapping[listener], cluster)

      clusters = append(clusters, u.createCluster(cluster))
      endpoints = append(endpoints, u.createEndpoint(cluster))
    }


    for listener, clusters := range resourceMapping {
      routes = append(routes, u.createRoute(listener, clusters))
    }
    u.grpcListeners = listeners
    u.grpcClusters = clusters
    u.grpcEndpoints = endpoints
    u.grpcRoutes = routes
}

func (u *Updater) Stop() {
  go func() {
    cmd := &cmd{
      command: "quit",
    }
    u.CmdChan <- *cmd
  }()
}

func (u *Updater) RefreshConfig(data []interface{}) {
  go func() {
    cmd := &cmd{
      command: "refresh",
      data: data,
    }
    u.CmdChan <- *cmd
  }()
}
