package lib

import (
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
)

type ServiceHost struct {
	Service     string
	IPAddress   string
	Port        int
	Tags        []string
	CreateIndex uint64
	ModifyIndex uint64
}

func (h ServiceHost) LbEndpoint() envoy_api_v2_endpoint.LbEndpoint {
	return envoy_api_v2_endpoint.LbEndpoint{
		HealthStatus: envoy_api_v2_core.HealthStatus_HEALTHY,
		Endpoint: &envoy_api_v2_endpoint.Endpoint{
			Address: &envoy_api_v2_core.Address{
				Address: &envoy_api_v2_core.Address_SocketAddress{
					SocketAddress: &envoy_api_v2_core.SocketAddress{
						Protocol: envoy_api_v2_core.TCP,
						Address:  h.IPAddress,
						PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
							PortValue: uint32(h.Port),
						},
					},
				},
			}}}
}

func NewServiceHost(endpoint *endpoint) ServiceHost {
	return ServiceHost{
		IPAddress: endpoint.address,
		Port:      endpoint.port,
	}
}
