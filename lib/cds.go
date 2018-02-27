package lib

import (
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

type Cluster interface {
	Cluster() *envoy_api_v2.Cluster
}

type cds struct {
	cluster ClusterConfig
	ads     bool
}

func (s *cds) clusterName() string {
	return s.cluster.Name
}

func (s *cds) configSource() *envoy_api_v2_core.ConfigSource {
	var edsSource *envoy_api_v2_core.ConfigSource
	if s.ads {
		edsSource = &envoy_api_v2_core.ConfigSource{
			ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_Ads{
				Ads: &envoy_api_v2_core.AggregatedConfigSource{},
			},
		}
	} else {
		edsSource = &envoy_api_v2_core.ConfigSource{
			ConfigSourceSpecifier: &envoy_api_v2_core.ConfigSource_ApiConfigSource{
				ApiConfigSource: &envoy_api_v2_core.ApiConfigSource{
					ApiType:      envoy_api_v2_core.ApiConfigSource_GRPC,
					ClusterNames: []string{"xds_cluster"},
				},
			},
		}
	}

	return edsSource
}

func (s *cds) configTLS() *envoy_api_v2_auth.UpstreamTlsContext {
	var tlsContext *envoy_api_v2_auth.UpstreamTlsContext
	if s.cluster.TLS == true {
		tlsContext = &envoy_api_v2_auth.UpstreamTlsContext{}
	}

	return tlsContext
}

func (s *cds) configHTTP2ProtoOpts() *envoy_api_v2_core.Http2ProtocolOptions {
	var http2ProtoOpts = &envoy_api_v2_core.Http2ProtocolOptions{}
	return http2ProtoOpts
}

func (s *cds) Cluster() *envoy_api_v2.Cluster {
	cluster := &envoy_api_v2.Cluster{
		Name:           s.clusterName(),
		ConnectTimeout: 5 * time.Second,
		Type:           envoy_api_v2.Cluster_EDS,
		EdsClusterConfig: &envoy_api_v2.Cluster_EdsClusterConfig{
			EdsConfig:   s.configSource(),
			ServiceName: s.clusterName(),
		},
		TlsContext: s.configTLS(),
	}

	if s.cluster.Protocol == "http2" {
		cluster.Http2ProtocolOptions = s.configHTTP2ProtoOpts()
	}
	cluster.HealthChecks = s.cluster.Healthchecks

	return cluster
}

func NewCluster(cluster ClusterConfig, ads bool) Cluster {
	return &cds{cluster: cluster, ads: ads}
}
