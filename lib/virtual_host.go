package lib

import (
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"

	"fmt"
)

type VirtualHost struct {
	Cluster ClusterConfig
}

func (v VirtualHost) getPrefix() string {
	if prefix := v.Cluster.Prefix; prefix == "" {
		return fmt.Sprintf("/%s", v.Cluster.Name)
	} else {
		return "/"
	}
}

func (v VirtualHost) VirtualHost() envoy_api_v2_route.VirtualHost {
	hashPolicy := []*envoy_api_v2_route.RouteAction_HashPolicy{}

	if hp := v.Cluster.Hashpolicy; hp != nil {
		hashPolicy = hp
	}

	return envoy_api_v2_route.VirtualHost{
		Name:    v.Cluster.Name,
		Domains: v.Cluster.Domains,
		Routes: []envoy_api_v2_route.Route{{
			Match: envoy_api_v2_route.RouteMatch{
				PathSpecifier: &envoy_api_v2_route.RouteMatch_Prefix{
					Prefix: v.getPrefix(),
				},
			},
			Action: &envoy_api_v2_route.Route_Route{
				Route: &envoy_api_v2_route.RouteAction{
					ClusterSpecifier: &envoy_api_v2_route.RouteAction_Cluster{
						Cluster: v.Cluster.Name,
					},
					HashPolicy: hashPolicy,
				},
			},
			Decorator: &envoy_api_v2_route.Decorator{
				Operation: v.Cluster.Name,
			},
		}},
	}
}

//NewServiceHost creates a new service host from a consul catalog service
func NewVirtualHost(cluster ClusterConfig) VirtualHost {
	return VirtualHost{
		Cluster: cluster,
	}
}
