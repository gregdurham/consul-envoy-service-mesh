package lib

import (
  cp "github.com/envoyproxy/go-control-plane/api"
  strConfig "github.com/gregdurham/consul-envoy-xds/config"
)

type Route interface {
  RouteCfg() *cp.RouteConfiguration
}

type route struct {
  name  string
  services []strConfig.Cluster
}

func (s *route) getVirtualHosts() []*cp.VirtualHost {
  hosts := make([]*cp.VirtualHost, 0)
  for _, cluster := range s.services {
    hosts = append(hosts, NewVirtualHost(cluster).VirtualHost())
  }
  return hosts
}

func (s * route) RouteCfg() *cp.RouteConfiguration {
  return &cp.RouteConfiguration{Name: s.name, VirtualHosts: s.getVirtualHosts()}
}

func NewRoute(name string, services []strConfig.Cluster) Route {
  return &route{name: name, services: services}
}

