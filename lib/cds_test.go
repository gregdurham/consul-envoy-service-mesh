package lib

import (
	"encoding/json"
	"errors"
	"testing"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"

	"github.com/stretchr/testify/assert"
)

func UnmarshalJson(data []byte) (*ClusterConfig, error) {
	var v Configs
	var c *ClusterConfig
	if err := json.Unmarshal(data, &v); err != nil {
		return c, err
	}

	for _, config := range v {
		if cluster, ok := config.(*ClusterConfig); ok {
			return cluster, nil
		} else {
			return c, errors.New("Not a cluster")
		}
	}
	return c, errors.New("Not a cluster")
}

func TestShouldHaveBasicSettings(t *testing.T) {

	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": false,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/"
	}`)
	cluster, err := UnmarshalJson(data)
	if err != nil {
		t.Fail()
	}
	envoyCluster := NewCluster(*cluster, true).Cluster()
	assert.Equal(t, "test_cluster", envoyCluster.Name)
	assert.Equal(t, float64(5), envoyCluster.ConnectTimeout.Seconds())
	assert.Equal(t, envoy_api_v2.Cluster_EDS, envoyCluster.Type)
	assert.Equal(t, "test_cluster", envoyCluster.EdsClusterConfig.ServiceName)
	var http2ProtocolOpts *envoy_api_v2_core.Http2ProtocolOptions
	assert.Equal(t, http2ProtocolOpts, envoyCluster.Http2ProtocolOptions)
	var tlsContext *envoy_api_v2_auth.UpstreamTlsContext
	assert.Equal(t, tlsContext, envoyCluster.TlsContext)
}

func TestShouldHaveBasicSettingsWithHTTP2(t *testing.T) {

	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": false,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/",
		"protocol": "http2"
	}`)
	cluster, err := UnmarshalJson(data)
	if err != nil {
		t.Fail()
	}
	envoyCluster := NewCluster(*cluster, true).Cluster()
	assert.Equal(t, &envoy_api_v2_core.Http2ProtocolOptions{}, envoyCluster.Http2ProtocolOptions)
}

func TestShouldHaveBasicSettingsWithTLS(t *testing.T) {

	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": true,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/"
	}`)
	cluster, err := UnmarshalJson(data)
	if err != nil {
		t.Fail()
	}
	envoyCluster := NewCluster(*cluster, true).Cluster()
	assert.Equal(t, &envoy_api_v2_auth.UpstreamTlsContext{}, envoyCluster.TlsContext)
}

func TestShouldHaveClusterWithHttpHealthCheck(t *testing.T) {

	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": true,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/",
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
	cluster, err := UnmarshalJson(data)
	if err != nil {
		t.Fail()
	}
	envoyCluster := NewCluster(*cluster, true).Cluster()
	hc := envoyCluster.HealthChecks[0]
	assert.Equal(t, int32(1000000000), hc.Timeout.Nanos)
	assert.Equal(t, int32(1000000000), hc.Interval.Nanos)
	assert.Equal(t, uint32(6), hc.HealthyThreshold.GetValue())
	assert.Equal(t, uint32(5), hc.UnhealthyThreshold.GetValue())
	assert.Equal(t, "/status", hc.GetHttpHealthCheck().Path)
}

func TestShouldHaveClusterWithTCPHealthCheck(t *testing.T) {

	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": true,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/",
		"health_checks": [
		  {
			"type": "tcp",
			"timeout_ms": 1000,
			"interval_ms": 1000,
			"unhealthy_threshold": 5,
			"healthy_threshold": 6,
			"send": ""
		  }
		]
	}`)
	cluster, err := UnmarshalJson(data)
	if err != nil {
		t.Fail()
	}
	envoyCluster := NewCluster(*cluster, true).Cluster()
	hc := envoyCluster.HealthChecks[0]
	assert.Equal(t, int32(1000000000), hc.Timeout.Nanos)
	assert.Equal(t, int32(1000000000), hc.Interval.Nanos)
	assert.Equal(t, uint32(6), hc.HealthyThreshold.GetValue())
	assert.Equal(t, uint32(5), hc.UnhealthyThreshold.GetValue())
	assert.Equal(t, "text:\"\" ", hc.GetTcpHealthCheck().Send.String())
}

func TestShouldHaveClusterWithRedisHealthCheck(t *testing.T) {

	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": true,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/",
		"health_checks": [
		  {
			"type": "redis",
			"timeout_ms": 1000,
			"interval_ms": 1000,
			"unhealthy_threshold": 5,
			"healthy_threshold": 6
		  }
		]
	}`)
	cluster, err := UnmarshalJson(data)
	if err != nil {
		t.Fail()
	}
	envoyCluster := NewCluster(*cluster, true).Cluster()
	hc := envoyCluster.HealthChecks[0]
	assert.Equal(t, int32(1000000000), hc.Timeout.Nanos)
	assert.Equal(t, int32(1000000000), hc.Interval.Nanos)
	assert.Equal(t, uint32(6), hc.HealthyThreshold.GetValue())
	assert.Equal(t, uint32(5), hc.UnhealthyThreshold.GetValue())
	assert.Equal(t, "", hc.GetRedisHealthCheck().String())
}

func TestShouldHaveClusterWithGrpcHealthCheck(t *testing.T) {

	data := []byte(`{
		"type": "cluster",
		"name": "test_cluster",
		"tls": true,
		"host": "127.0.0.1",
		"port": 8080,
		"domains": ["*"],
		"prefix": "/",
		"health_checks": [
		  {
			"type": "grpc",
			"timeout_ms": 1000,
			"interval_ms": 1000,
			"unhealthy_threshold": 5,
			"healthy_threshold": 6
		  }
		]
	}`)
	cluster, err := UnmarshalJson(data)
	if err != nil {
		t.Fail()
	}
	envoyCluster := NewCluster(*cluster, true).Cluster()
	hc := envoyCluster.HealthChecks[0]
	assert.Equal(t, int32(1000000000), hc.Timeout.Nanos)
	assert.Equal(t, int32(1000000000), hc.Interval.Nanos)
	assert.Equal(t, uint32(6), hc.HealthyThreshold.GetValue())
	assert.Equal(t, uint32(5), hc.UnhealthyThreshold.GetValue())
	assert.Equal(t, "", hc.GetGrpcHealthCheck().String())
}
