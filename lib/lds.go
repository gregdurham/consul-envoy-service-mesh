package lib

import (
  strConfig "github.com/gregdurham/consul-envoy-xds/config"

  "github.com/envoyproxy/go-control-plane/api/filter/network"
  httpFilter "github.com/envoyproxy/go-control-plane/api/filter/http"
  accessLog "github.com/envoyproxy/go-control-plane/api/filter/accesslog"
  "github.com/envoyproxy/go-control-plane/pkg/util"
  cp "github.com/envoyproxy/go-control-plane/api"

  "github.com/gogo/protobuf/types"

  "path/filepath"
  "fmt"
)

type Listener interface {
  Listener() *cp.Listener
}

type lds struct {
  listener strConfig.Listener
  ads   bool
}

func (s *lds) Ads() bool {
  return s.ads
}

func (s *lds) configSource() cp.ConfigSource {
  var rdsSource cp.ConfigSource
  if s.Ads() {
    rdsSource.ConfigSourceSpecifier = &cp.ConfigSource_Ads{
      Ads: &cp.AggregatedConfigSource{},
    }
  } else {
    rdsSource.ConfigSourceSpecifier = &cp.ConfigSource_ApiConfigSource{
      ApiConfigSource: &cp.ApiConfigSource{
        ApiType:     cp.ApiConfigSource_GRPC,
        ClusterName: []string{"xds_cluster"},
      },
    }
  }
  return rdsSource
}

func (s *lds) healthCheck() *types.Struct {
  healthCheck := &httpFilter.HealthCheck{
    PassThroughMode: &types.BoolValue{true},
    Endpoint: "/status",
  }
  pbst, err := util.MessageToStruct(healthCheck)
  if err != nil {
    panic(err)
  }
  return pbst
}

func (s *lds) router() *types.Struct {
  routerConfig := &httpFilter.Router{}
  pbst, err := util.MessageToStruct(routerConfig)
  if err != nil {
    panic(err)
  }
  return pbst
}

func (s *lds) accessLogConfig(fileName string) *types.Struct {
  accessLogConfig := &accessLog.FileAccessLog{
    Path: fmt.Sprintf("/var/log/envoy/%s", fileName),
    Format: "[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\"\n",
  }
  pbst, err := util.MessageToStruct(accessLogConfig)
  if err != nil {
    panic(err)
  }
  return pbst
}

func (s *lds) errorLog(statPrefix string) *accessLog.AccessLog {
  return &accessLog.AccessLog{
    Name: "envoy.file_access_log",
    Filter: &accessLog.AccessLogFilter{
      FilterSpecifier: &accessLog.AccessLogFilter_AndFilter{
        AndFilter: &accessLog.AndFilter{
          Filters: []*accessLog.AccessLogFilter{{
            FilterSpecifier: &accessLog.AccessLogFilter_OrFilter{
              OrFilter: &accessLog.OrFilter{
                Filters: []*accessLog.AccessLogFilter{{
                  FilterSpecifier: &accessLog.AccessLogFilter_StatusCodeFilter{
                    StatusCodeFilter: &accessLog.StatusCodeFilter{
                      Comparison: &accessLog.ComparisonFilter{
                        Op: accessLog.ComparisonFilter_GE,
                        Value: &cp.RuntimeUInt32{
                          DefaultValue: uint32(200),
                          RuntimeKey: "test",
                        },
                      },
                    },
                  },
                },{
                  FilterSpecifier: &accessLog.AccessLogFilter_StatusCodeFilter{
                    StatusCodeFilter: &accessLog.StatusCodeFilter{
                      Comparison: &accessLog.ComparisonFilter{
                        Op: accessLog.ComparisonFilter_EQ,
                        Value: &cp.RuntimeUInt32{
                          DefaultValue: uint32(0),
                          RuntimeKey: "test",
                        },
                      },
                    },
                  },
                },{
                  FilterSpecifier: &accessLog.AccessLogFilter_DurationFilter{
                    DurationFilter: &accessLog.DurationFilter{
                      Comparison: &accessLog.ComparisonFilter{
                        Op: accessLog.ComparisonFilter_GE,
                        Value: &cp.RuntimeUInt32{
                          DefaultValue: uint32(2000),
                          RuntimeKey: "test",
                        },
                      },
                    },
                  },
                }},
              },
            },
          },{
            FilterSpecifier: &accessLog.AccessLogFilter_NotHealthCheckFilter{
              NotHealthCheckFilter: &accessLog.NotHealthCheckFilter{},
            },
          }},
        },
      },
    },
    Config: s.accessLogConfig(fmt.Sprintf("%s_error.log", statPrefix)),
  }
}

func (s *lds) accessLog(statPrefix string) *accessLog.AccessLog {
  return &accessLog.AccessLog{
    Name: "envoy.file_access_log",
    Filter: &accessLog.AccessLogFilter{
      FilterSpecifier: &accessLog.AccessLogFilter_NotHealthCheckFilter{
        NotHealthCheckFilter: &accessLog.NotHealthCheckFilter{},
      },
    },
    Config: s.accessLogConfig(fmt.Sprintf("%s.log", statPrefix)),
  }
}

func (s *lds) manager() *types.Struct {
  logs := []*accessLog.AccessLog{}
  operation := network.HttpConnectionManager_Tracing_EGRESS
  httpFilters := []*network.HttpFilter{{
    Name: "envoy.router",
    Config: s.router(),
  }}

  statPrefix := ""

  if s.listener.GetName() == "local_service" {
    statPrefix = "ingress_http"
    operation = network.HttpConnectionManager_Tracing_INGRESS
    healthCheck := &network.HttpFilter{
      Name: "envoy.health_check",
      Config: s.healthCheck(),
    }

    httpFilters = append(httpFilters, healthCheck)
    logs = append(logs, s.accessLog(statPrefix))
    logs = append(logs, s.errorLog(statPrefix))
  } else {
    statPrefix = "egress_http"
    logs = append(logs, s.accessLog(statPrefix))
  }
  
  manager := &network.HttpConnectionManager{
    CodecType:  network.HttpConnectionManager_AUTO,
    StatPrefix: statPrefix,
    Tracing: &network.HttpConnectionManager_Tracing{
      OperationName: operation,
    },
    RouteSpecifier: &network.HttpConnectionManager_Rds{
      Rds: &network.Rds{
        ConfigSource:    s.configSource(),
        RouteConfigName: s.listener.GetName(),
      },
    },
    HttpFilters: httpFilters,
    AccessLog: logs,
  }
  pbst, err := util.MessageToStruct(manager)
  if err != nil {
    panic(err)
  }
  return pbst
}

func (s *lds) configTLS() *cp.DownstreamTlsContext {
  const base = "/etc/envoy/conf.d/"
  return &cp.DownstreamTlsContext{
    CommonTlsContext: &cp.CommonTlsContext{
      TlsCertificates: []*cp.TlsCertificate{{
        CertificateChain: &cp.DataSource{
          &cp.DataSource_Filename{
            Filename: filepath.Join(base, "cert.pem"),
          },
        },
        PrivateKey: &cp.DataSource{
          &cp.DataSource_Filename{
            Filename: filepath.Join(base, "priv.pem"),
          },
        },
      }},
    },
  }
}

func (s *lds) configFilterChain() *cp.FilterChain {
  var filterChain *cp.FilterChain
  if s.listener.GetTLS() == true {
    filterChain = &cp.FilterChain{
      TlsContext: s.configTLS(),
      Filters: []*cp.Filter{{
        Name:   "envoy.http_connection_manager",
        Config: s.manager(),
      }},
    }
  } else {
    filterChain = &cp.FilterChain{
      Filters: []*cp.Filter{{
        Name:   "envoy.http_connection_manager",
        Config: s.manager(),
      }},
    }
  }
  return filterChain
}

func (s *lds) Listener() *cp.Listener {
  listener := &cp.Listener{
    Name: s.listener.GetName(),
    Address: &cp.Address{
      Address: &cp.Address_SocketAddress{
        SocketAddress: &cp.SocketAddress{
          Protocol: cp.SocketAddress_TCP,
          Address:  s.listener.GetHost(),
          PortSpecifier: &cp.SocketAddress_PortValue{
            PortValue: uint32(s.listener.GetPort()),
          },
        },
      },
    },
    FilterChains: []*cp.FilterChain{},
  }

  listener.FilterChains = append(listener.FilterChains, s.configFilterChain())
  return listener
}

func NewListener(listener strConfig.Listener, ads bool) Listener {
  return &lds{listener: listener,
                  ads: ads}
}
