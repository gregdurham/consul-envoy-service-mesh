package lib

import (
	"encoding/json"
	"testing"

	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/stretchr/testify/assert"
)

func TestShouldHaveRouteWithVirtualHost(t *testing.T) {
	var v Configs
	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": false,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/",
		"protocol": "http2",
		"health_checks": [
		  {
			"type": "http",
			"timeout_ms": 1000,
			"interval_ms": 1000,
			"unhealthy_threshold": 5,
			"healthy_threshold": 6,
			"path": "/status"
		  }
		]
	}`)
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fail()
	}
	clusters := make([]ClusterConfig, 0)
	for _, config := range v {
		if cluster, ok := config.(*ClusterConfig); ok {
			clusters = append(clusters, *cluster)
			rt := NewRoute("test", clusters)
			rtConfig := rt.RouteCfg()
			assert.Equal(t, "test", rtConfig.Name)
			vhost := rtConfig.VirtualHosts[0]
			assert.Equal(t, "test_cluster", vhost.Name)
			assert.Equal(t, []string{"*"}, vhost.Domains)
			route := vhost.Routes[0]
			assert.Equal(t, &envoy_api_v2_route.RouteMatch_Prefix{Prefix: "/"}, route.Match.PathSpecifier)
		} else {
			t.Fail()
		}
	}

}

/*
func TestShouldHaveRouteWithVirtualHost(t *testing.T) {
	x := []*ServiceEndpoint{{address: "127.0.0.1", port: 8080}}
	endpoint := NewEndpoint("test", x)
	cla := endpoint.CLA()
	t.Log(cla)
	assert.Equal(t, "test", cla.ClusterName)
	localityEndpoint := cla.Endpoints[0]
	assert.Equal(t, "socket_address:<address:\"127.0.0.1\" port_value:8080 > ", localityEndpoint.LbEndpoints[0].Endpoint.Address.String())
}
*/
