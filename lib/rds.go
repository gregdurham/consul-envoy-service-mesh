package lib

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
)

type Route interface {
	RouteCfg() *envoy_api_v2.RouteConfiguration
}

type route struct {
	name     string
	services []ClusterConfig
}

func (s *route) getVirtualHosts() []envoy_api_v2_route.VirtualHost {
	hosts := make([]envoy_api_v2_route.VirtualHost, 0)
	for _, cluster := range s.services {
		hosts = append(hosts, NewVirtualHost(cluster).VirtualHost())
	}
	return hosts
}

func (s *route) RouteCfg() *envoy_api_v2.RouteConfiguration {
	return &envoy_api_v2.RouteConfiguration{Name: s.name, VirtualHosts: s.getVirtualHosts()}
}

func NewRoute(name string, services []ClusterConfig) Route {
	return &route{name: name, services: services}
}
