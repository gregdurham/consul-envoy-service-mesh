package lib

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
)

type Endpoint interface {
	CLA() *v2.ClusterLoadAssignment
}

type eds struct {
	clusterName string
	endpoints   []*endpoint
}

func (s *eds) getLbEndpoints() []envoy_api_v2_endpoint.LbEndpoint {
	hosts := make([]envoy_api_v2_endpoint.LbEndpoint, 0)
	for _, s := range s.endpoints {
		hosts = append(hosts, NewServiceHost(s).LbEndpoint())
	}
	return hosts
}

//Need to override
func (s *eds) locality() *envoy_api_v2_core.Locality {
	return &envoy_api_v2_core.Locality{
		Region: "dc1",
	}
}

func (s *eds) getLocalityEndpoints() []envoy_api_v2_endpoint.LocalityLbEndpoints {
	return []envoy_api_v2_endpoint.LocalityLbEndpoints{{Locality: s.locality(), LbEndpoints: s.getLbEndpoints()}}
}

func (s *eds) claPolicy() *envoy_api_v2.ClusterLoadAssignment_Policy {
	return &envoy_api_v2.ClusterLoadAssignment_Policy{DropOverload: 0.0}
}

func (s *eds) CLA() *v2.ClusterLoadAssignment {
	return &v2.ClusterLoadAssignment{Endpoints: s.getLocalityEndpoints(), ClusterName: s.clusterName, Policy: s.claPolicy()}
}

//NewEndpoint creates an ServiceEndpoint representation
func NewEndpoint(clusterName string, endpoints []*endpoint) Endpoint {
	return &eds{
		clusterName: clusterName,
		endpoints:   endpoints,
	}
}
