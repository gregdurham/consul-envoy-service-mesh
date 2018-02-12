package lib

import (
  strConfig "github.com/gregdurham/consul-envoy-service-mesh/config"
  envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
  
  "fmt"
)

type VirtualHost struct {
  Cluster strConfig.Cluster
}

func (v VirtualHost) getPrefix() string {
  if prefix := v.Cluster.GetPrefix(); prefix == "" {
    return fmt.Sprintf("/%s", v.Cluster.GetName())
  } else {
    return "/"
  }
}

func (v VirtualHost) CookieHashPolicy() *envoy_api_v2_route.RouteAction_HashPolicy {
  return &envoy_api_v2_route.RouteAction_HashPolicy{
    PolicySpecifier: &envoy_api_v2_route.RouteAction_HashPolicy_Cookie_{
      Cookie: &envoy_api_v2_route.RouteAction_HashPolicy_Cookie{
        Name: v.Cluster.GetCookie(),
      },
    },
  }
}

func (v VirtualHost) HeaderHashPolicy() *envoy_api_v2_route.RouteAction_HashPolicy {
  return &envoy_api_v2_route.RouteAction_HashPolicy{
    PolicySpecifier: &envoy_api_v2_route.RouteAction_HashPolicy_Header_{
      Header: &envoy_api_v2_route.RouteAction_HashPolicy_Header{
        HeaderName: v.Cluster.GetHeader(),
      },
    },
  }
}

func (v VirtualHost) ConnectionPropertiesHashPolicy() *envoy_api_v2_route.RouteAction_HashPolicy {
  return &envoy_api_v2_route.RouteAction_HashPolicy{
    PolicySpecifier: &envoy_api_v2_route.RouteAction_HashPolicy_ConnectionProperties_{
      ConnectionProperties: &envoy_api_v2_route.RouteAction_HashPolicy_ConnectionProperties{
        SourceIp: v.Cluster.GetSourceip(),
      },
    },
  }
}

func (v VirtualHost) VirtualHost() envoy_api_v2_route.VirtualHost {
  hashPolicy := []*envoy_api_v2_route.RouteAction_HashPolicy{}

  for _, policy := range v.Cluster.GetHashpolicy() {
    if policy == "cookie" {
      hashPolicy = append(hashPolicy, v.CookieHashPolicy())
    } else if policy == "header" {
      hashPolicy = append(hashPolicy, v.HeaderHashPolicy())
    } else if policy == "ConnectionProperties" {
      hashPolicy = append(hashPolicy, v.ConnectionPropertiesHashPolicy())
    }
  }

  return envoy_api_v2_route.VirtualHost{
    Name:    v.Cluster.GetName(),
    Domains: v.Cluster.GetDomains(),
    Routes: []envoy_api_v2_route.Route{{
      Match: envoy_api_v2_route.RouteMatch{
        PathSpecifier: &envoy_api_v2_route.RouteMatch_Prefix{
          Prefix: v.getPrefix(),
        },
      },
      Action: &envoy_api_v2_route.Route_Route{
        Route: &envoy_api_v2_route.RouteAction{
          ClusterSpecifier: &envoy_api_v2_route.RouteAction_Cluster{
            Cluster: v.Cluster.GetName(),
          },
          HashPolicy: hashPolicy,
        },
      },
      Decorator: &envoy_api_v2_route.Decorator{
        Operation: v.Cluster.GetName(),
      },
    }},
  }
}

//NewServiceHost creates a new service host from a consul catalog service
func NewVirtualHost(cluster strConfig.Cluster) VirtualHost {
  return VirtualHost{
    Cluster: cluster,
  }
}
