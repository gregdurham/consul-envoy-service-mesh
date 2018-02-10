package lib

import (
  strConfig "github.com/gregdurham/consul-envoy-xds/config"
  cp "github.com/envoyproxy/go-control-plane/api"
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

func (v VirtualHost) CookieHashPolicy() *cp.RouteAction_HashPolicy {
  return &cp.RouteAction_HashPolicy{
    PolicySpecifier: &cp.RouteAction_HashPolicy_Cookie_{
      Cookie: &cp.RouteAction_HashPolicy_Cookie{
        Name: v.Cluster.GetCookie(),
      },
    },
  }
}

func (v VirtualHost) HeaderHashPolicy() *cp.RouteAction_HashPolicy {
  return &cp.RouteAction_HashPolicy{
    PolicySpecifier: &cp.RouteAction_HashPolicy_Header_{
      Header: &cp.RouteAction_HashPolicy_Header{
        HeaderName: v.Cluster.GetHeader(),
      },
    },
  }
}

func (v VirtualHost) ConnectionPropertiesHashPolicy() *cp.RouteAction_HashPolicy {
  return &cp.RouteAction_HashPolicy{
    PolicySpecifier: &cp.RouteAction_HashPolicy_ConnectionProperties_{
      ConnectionProperties: &cp.RouteAction_HashPolicy_ConnectionProperties{
        SourceIp: v.Cluster.GetSourceip(),
      },
    },
  }
}

func (v VirtualHost) VirtualHost() *cp.VirtualHost {
  hashPolicy := []*cp.RouteAction_HashPolicy{}

  for _, policy := range v.Cluster.GetHashpolicy() {
    if policy == "cookie" {
      hashPolicy = append(hashPolicy, v.CookieHashPolicy())
    } else if policy == "header" {
      hashPolicy = append(hashPolicy, v.HeaderHashPolicy())
    } else if policy == "ConnectionProperties" {
      hashPolicy = append(hashPolicy, v.ConnectionPropertiesHashPolicy())
    }
  }

  return &cp.VirtualHost{
    Name:    v.Cluster.GetName(),
    Domains: v.Cluster.GetDomains(),
    Routes: []*cp.Route{{
      Match: &cp.RouteMatch{
        PathSpecifier: &cp.RouteMatch_Prefix{
          Prefix: v.getPrefix(),
        },
      },
      Action: &cp.Route_Route{
        Route: &cp.RouteAction{
          ClusterSpecifier: &cp.RouteAction_Cluster{
            Cluster: v.Cluster.GetName(),
          },
          HashPolicy: hashPolicy,
        },
      },
      Decorator: &cp.Decorator{
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
