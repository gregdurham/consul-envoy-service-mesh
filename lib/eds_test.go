package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldHaveCLAUsingServiceEndpoint(t *testing.T) {
	x := []*endpoint{{address: "127.0.0.1", port: 8080}}
	endpoint := NewEndpoint("test", x)
	cla := endpoint.CLA()
	assert.Equal(t, "test", cla.ClusterName)
	localityEndpoint := cla.Endpoints[0]
	assert.Equal(t, "socket_address:<address:\"127.0.0.1\" port_value:8080 > ", localityEndpoint.LbEndpoints[0].Endpoint.Address.String())
}

func TestShouldHaveCLAUsingMultipleServiceEndpoints(t *testing.T) {
	x := []*endpoint{{address: "127.0.0.1", port: 8080}, {address: "127.0.0.1", port: 8081}}
	endpoint := NewEndpoint("test", x)
	cla := endpoint.CLA()
	assert.Equal(t, "test", cla.ClusterName)
	localityEndpoint := cla.Endpoints[0]
	assert.Equal(t, "socket_address:<address:\"127.0.0.1\" port_value:8080 > ", localityEndpoint.LbEndpoints[0].Endpoint.Address.String())
	assert.Equal(t, "socket_address:<address:\"127.0.0.1\" port_value:8081 > ", localityEndpoint.LbEndpoints[1].Endpoint.Address.String())
}
