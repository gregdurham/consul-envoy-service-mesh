package lib

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/cskr/pubsub"
	cpCache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/golang/glog"
)

type Updater struct {
	ID               string
	Listeners        []ListenerConfig
	Clusters         []ClusterConfig
	Subscriptions    []string
	ServiceEndpoints EndpointIndex
	Cache            cpCache.SnapshotCache
	Events           *pubsub.PubSub
	CmdChan          chan cmd
	ErrorChan        chan error
	SnapshotId       int
	grpcListeners    []cpCache.Resource
	grpcClusters     []cpCache.Resource
	grpcEndpoints    []cpCache.Resource
	grpcRoutes       []cpCache.Resource
}

type endpoint struct {
	address string
	port    int
}

type cmd struct {
	command string
	data    Resource
	reply   chan []string
}

func NewUpdater(id string, cache cpCache.SnapshotCache, events *pubsub.PubSub, serviceEndpoints EndpointIndex, errorChan chan error) Updater {
	updater := Updater{
		ID:               id,
		Cache:            cache,
		Events:           events,
		ServiceEndpoints: serviceEndpoints,
		CmdChan:          make(chan cmd),
		ErrorChan:        errorChan,
		SnapshotId:       0,
	}

	return updater
}

func (u *Updater) SetResources(resource Resource) {
	listeners := resource.Listeners
	clusters := resource.Clusters
	subscriptions := []string{}
	for _, cluster := range resource.Clusters {
		if cluster.Host == "" && cluster.Port == 0 {
			subscriptions = append(subscriptions, cluster.Name)
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
	u.writeSnapshot()
	go func() {
		for {
			select {
			case cnc := <-u.CmdChan:
				if cnc.command == "quit" {
					glog.V(10).Infof("Received request to quit %s", u.ID)
					return
				} else if cnc.command == "refresh" {
					if !reflect.DeepEqual(cnc.data, Resource{Clusters: u.Clusters, Listeners: u.Listeners}) {
						glog.V(10).Infof("Received request to refresh configuration %s", u.ID)
						u.SetResources(cnc.data)
						subs := []string{}
						u.Events.Unsub(SubChannel, subs...)
						u.updateEndpoint()
						SubChannel = u.Events.Sub(u.Subscriptions...)
						u.writeSnapshot()
					}
				} else if cnc.command == "getSubscriptions" {
					cnc.reply <- u.Subscriptions
				}
			case <-SubChannel:
				u.updateEndpoint()
				u.writeSnapshot()
			}
		}
	}()
}

func (u *Updater) updateEndpoint() {
	glog.V(10).Infof("Refreshing endpoints")
	endpoints := make([]cpCache.Resource, 0)
	for _, cluster := range u.Clusters {
		endpoints = append(endpoints, u.createEndpoint(cluster))
	}

	u.grpcEndpoints = endpoints
}

func (u *Updater) writeSnapshot() {
	glog.V(10).Infof("Write snapshot called, version is %d", u.SnapshotId)
	version := fmt.Sprintf("version%d", u.SnapshotId)
	snapshot := cpCache.NewSnapshot(version,
		u.grpcEndpoints,
		u.grpcClusters,
		u.grpcRoutes,
		u.grpcListeners)
	u.Cache.SetSnapshot(u.ID, snapshot)
	u.SnapshotId++
}

func (u *Updater) createListener(listener ListenerConfig) cpCache.Resource {
	lstn := NewListener(listener, true)
	return lstn.Listener()
}

func (u *Updater) createCluster(cluster ClusterConfig) cpCache.Resource {
	cls := NewCluster(cluster, true)
	return cls.Cluster()
}

func (u *Updater) createEndpoint(cluster ClusterConfig) cpCache.Resource {
	if cluster.Host != "" && cluster.Port != 0 {
		svc := []*endpoint{{
			address: cluster.Host,
			port:    cluster.Port,
		}}
		ep := NewEndpoint(cluster.Name, svc)
		return ep.CLA()
	}
	if k, ok := u.ServiceEndpoints.GetEndpoints(cluster.Name); ok {
		ep := NewEndpoint(cluster.Name, k)
		return ep.CLA()
	}
	return nil
}

func (u *Updater) createRoute(listener string, clusters []ClusterConfig) cpCache.Resource {
	rt := NewRoute(listener, clusters)
	return rt.RouteCfg()
}

func (u *Updater) updateConfiguration() {
	listeners := make([]cpCache.Resource, 0)
	clusters := make([]cpCache.Resource, 0)
	endpoints := make([]cpCache.Resource, 0)
	routes := make([]cpCache.Resource, 0)

	resourceMapping := make(map[string][]ClusterConfig, 0)

	for _, listener := range u.Listeners {
		listeners = append(listeners, u.createListener(listener))
		for _, clusterName := range listener.Clusters {
			for _, cluster := range u.Clusters {
				if clusterName == cluster.Name {
					resourceMapping[listener.Name] = append(resourceMapping[listener.Name], cluster)

					clusters = append(clusters, u.createCluster(cluster))
					endpoints = append(endpoints, u.createEndpoint(cluster))
				}
			}
		}
	}

	for listener, clusters := range resourceMapping {
		routes = append(routes, u.createRoute(listener, clusters))
	}
	//u.printJson(listeners)
	//u.printJson(clusters)
	//u.printJson(endpoints)
	//u.printJson(routes)
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

func (u *Updater) RefreshConfig(data Resource) {
	go func() {
		cmd := &cmd{
			command: "refresh",
			data:    data,
		}
		u.CmdChan <- *cmd
	}()
}

func (u *Updater) printJson(resource []cpCache.Resource) {
	out, err := json.Marshal(resource)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))
}

func (u *Updater) GetSubscriptions() ([]string, error) {
	subscriptions := make(chan []string, 1)
	cmd := &cmd{
		command: "getSubscriptions",
		reply:   subscriptions,
	}
	u.CmdChan <- *cmd
	return <-subscriptions, nil
}

/*
func (u *Updater) GetSubscriptions() ([]string, error) {
	subscriptions := make(chan []string, 1)
	go func() {
		cmd := &cmd{
			command: "getSubscriptions",
			reply:   subscriptions,
		}
		u.CmdChan <- *cmd
	}()
	return <-subscriptions, nil
}
*/
/*
func (u *Updater) GetSubscriptions() ([]string, error) {
	temp := make(chan []string, 1)
	go func(temp chan []string) {
		subscriptions := make(chan []string, 1)
		cmd := &cmd{
			command: "getSubscriptions",
			reply:   subscriptions,
		}
		u.CmdChan <- *cmd
		x := <-subscriptions
		temp <- x
	}(temp)
	return <-temp, nil
}*/

/*
func (u *Updater) GetSubscriptions() ([]string, error) {
	temp := make(chan []string, 0)
	go func(temp chan []string) {
		subscriptions := make(chan []string, 0)
		cmd := &cmd{
			command: "getSubscriptions",
			reply:   subscriptions,
		}
		u.CmdChan <- *cmd
		x := <-subscriptions
		temp <- x
	}(temp)
	return <-temp, nil
}*/
/*
func (u *Updater) GetSubscriptions(subscriptions chan []string) {
	go func() {
		cmd := &cmd{
			command: "getSubscriptions",
			reply:   subscriptions,
		}
		u.CmdChan <- *cmd
	}()
}*/
