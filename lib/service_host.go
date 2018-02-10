package lib

import (
  cp "github.com/envoyproxy/go-control-plane/api"
  "github.com/hashicorp/consul/api"
)

type ServiceHost struct {
  Service     string
  IPAddress   string
  Port        int
  Tags        []string
  CreateIndex uint64
  ModifyIndex uint64
}

func (h ServiceHost) LbEndpoint() *cp.LbEndpoint {
  return &cp.LbEndpoint{
    HealthStatus: cp.HealthStatus_HEALTHY,
    Endpoint: &cp.Endpoint{
      Address: &cp.Address{
        Address: &cp.Address_SocketAddress{
          SocketAddress: &cp.SocketAddress{
            Protocol: cp.SocketAddress_TCP,
            Address:  h.IPAddress,
            PortSpecifier: &cp.SocketAddress_PortValue{
              PortValue: uint32(h.Port),
            },
          },
        },
      }}}
}

func NewServiceHostFromCatalog(s *api.CatalogService) ServiceHost {
  return ServiceHost{
    IPAddress:   s.ServiceAddress,
    Port:        s.ServicePort,
    Tags:        s.ServiceTags,
    Service:     s.ServiceName,
    CreateIndex: s.CreateIndex,
    ModifyIndex: s.ModifyIndex,
  }
}

func NewServiceHost(endpoint *endpoint) ServiceHost {
  return ServiceHost{
    IPAddress:   endpoint.address,
    Port:        endpoint.port,
  }
}