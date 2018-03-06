package lib

import "sync"

type Service struct {
	Name, ResourceType string
}

type ConfigMapping struct {
	M         sync.Mutex
	Cluster   map[string]ClusterConfig
	Listerner map[string]ListenerConfig
	Service   map[string]ServiceConfig
}

type Resource struct {
	Clusters  []ClusterConfig
	Listeners []ListenerConfig
}

type Resources map[string]Resource

func NewConfigMapping() *ConfigMapping {
	var c ConfigMapping
	c.Cluster = make(map[string]ClusterConfig)
	c.Listerner = make(map[string]ListenerConfig)
	c.Service = make(map[string]ServiceConfig)
	return &c
}

func NewResource() *Resource {
	var r Resource
	r.Clusters = make([]ClusterConfig, 0)
	r.Listeners = make([]ListenerConfig, 0)
	return &r
}
