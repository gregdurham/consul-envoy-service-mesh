package lib

import (
  "time"

  strConfig "github.com/gregdurham/consul-envoy-xds/config"

  cp "github.com/envoyproxy/go-control-plane/api"
)

type Cluster interface {
  Cluster() *cp.Cluster
}

type cds struct {
  cluster  strConfig.Cluster
  ads   bool
}

func (s *cds) clusterName() string {
  return s.cluster.GetName()
}

func (s *cds) configSource() *cp.ConfigSource {
  var edsSource *cp.ConfigSource
  if s.ads {
    edsSource = &cp.ConfigSource{
      ConfigSourceSpecifier: &cp.ConfigSource_Ads{
        Ads: &cp.AggregatedConfigSource{},
      },
    }
  } else {
    edsSource = &cp.ConfigSource{
      ConfigSourceSpecifier: &cp.ConfigSource_ApiConfigSource{
        ApiConfigSource: &cp.ApiConfigSource{
          ApiType:      cp.ApiConfigSource_GRPC,
          ClusterName: []string{"xds_cluster"},
        },
      },
    }
  }

  return edsSource
}

func (s *cds) configTLS() *cp.UpstreamTlsContext {
  var tlsContext *cp.UpstreamTlsContext
  if s.cluster.GetTLS() == true {
   tlsContext = &cp.UpstreamTlsContext{} 
  }

  return tlsContext
}

func (s *cds) Cluster() *cp.Cluster {
  return &cp.Cluster{
    Name:           s.clusterName(),
    ConnectTimeout: 5 * time.Second,
    Type:           cp.Cluster_EDS,
    EdsClusterConfig: &cp.Cluster_EdsClusterConfig{
      EdsConfig:   s.configSource(),
      ServiceName: s.clusterName(),
    },
    TlsContext: s.configTLS(),
  }
}

func NewCluster(cluster strConfig.Cluster, ads bool) Cluster {
  return &cds{cluster: cluster, ads: ads}
}
