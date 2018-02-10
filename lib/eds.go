package lib

import (
  cp "github.com/envoyproxy/go-control-plane/api"
)

type Endpoint interface {
  CLA() *cp.ClusterLoadAssignment
}

type eds struct {
  clusterName string
  endpoints   []*endpoint
}

func (s *eds) getLbEndpoints() []*cp.LbEndpoint {
  hosts := make([]*cp.LbEndpoint, 0)
  for _, s := range s.endpoints {
    hosts = append(hosts, NewServiceHost(s).LbEndpoint())
  }
  return hosts
}

//Need to override
func (s *eds) locality() *cp.Locality {
  return &cp.Locality{
    Region: "dc1",
  }
}

func (s *eds) getLocalityEndpoints() []*cp.LocalityLbEndpoints {
  return []*cp.LocalityLbEndpoints{{Locality: s.locality(), LbEndpoints: s.getLbEndpoints()}}
}


func (s *eds) claPolicy() *cp.ClusterLoadAssignment_Policy {
  return &cp.ClusterLoadAssignment_Policy{DropOverload: 0.0}
}

func (s *eds) CLA() *cp.ClusterLoadAssignment {
  return &cp.ClusterLoadAssignment{Endpoints: s.getLocalityEndpoints(), ClusterName: s.clusterName, Policy: s.claPolicy()}
}

//NewEndpoint creates an ServiceEndpoint representation
func NewEndpoint(clusterName string, endpoints []*endpoint) Endpoint {
  return &eds{
    clusterName: clusterName,
    endpoints: endpoints,
  }
}
