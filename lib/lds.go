package lib

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	envoy_config_filter_accesslog_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	envoy_config_filter_http_health_check_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/health_check/v2"
	envoy_config_filter_http_router_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/util"

	"github.com/gogo/protobuf/types"

	"fmt"
	"path/filepath"
)

type Listener interface {
	Listener() *envoy_api_v2.Listener
}

type lds struct {
	listener ListenerConfig
	ads      bool
}

func (s *lds) Ads() bool {
	return s.ads
}

func (s *lds) configSource() envoy_api_v2_core.ConfigSource {
	var rdsSource envoy_api_v2_core.ConfigSource
	if s.Ads() {
		rdsSource.ConfigSourceSpecifier = &envoy_api_v2_core.ConfigSource_Ads{
			Ads: &envoy_api_v2_core.AggregatedConfigSource{},
		}
	} else {
		rdsSource.ConfigSourceSpecifier = &envoy_api_v2_core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &envoy_api_v2_core.ApiConfigSource{
				ApiType:      envoy_api_v2_core.ApiConfigSource_GRPC,
				ClusterNames: []string{"xds_cluster"},
			},
		}
	}
	return rdsSource
}

func (s *lds) healthCheck(healthCheck *envoy_config_filter_http_health_check_v2.HealthCheck) *types.Struct {
	pbst, err := util.MessageToStruct(healthCheck)
	if err != nil {
		panic(err)
	}
	return pbst
}

func (s *lds) router() *types.Struct {
	routerConfig := &envoy_config_filter_http_router_v2.Router{}
	pbst, err := util.MessageToStruct(routerConfig)
	if err != nil {
		panic(err)
	}
	return pbst
}

func (s *lds) accessLogConfig(fileName string) *types.Struct {
	accessLogConfig := &envoy_config_filter_accesslog_v2.FileAccessLog{
		Path:   fmt.Sprintf("/var/log/envoy/%s", fileName),
		Format: "[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\"\n",
	}
	pbst, err := util.MessageToStruct(accessLogConfig)
	if err != nil {
		panic(err)
	}
	return pbst
}

func (s *lds) errorLog(statPrefix string) *envoy_config_filter_accesslog_v2.AccessLog {
	return &envoy_config_filter_accesslog_v2.AccessLog{
		Name: "envoy.file_access_log",
		Filter: &envoy_config_filter_accesslog_v2.AccessLogFilter{
			FilterSpecifier: &envoy_config_filter_accesslog_v2.AccessLogFilter_AndFilter{
				AndFilter: &envoy_config_filter_accesslog_v2.AndFilter{
					Filters: []*envoy_config_filter_accesslog_v2.AccessLogFilter{{
						FilterSpecifier: &envoy_config_filter_accesslog_v2.AccessLogFilter_OrFilter{
							OrFilter: &envoy_config_filter_accesslog_v2.OrFilter{
								Filters: []*envoy_config_filter_accesslog_v2.AccessLogFilter{{
									FilterSpecifier: &envoy_config_filter_accesslog_v2.AccessLogFilter_StatusCodeFilter{
										StatusCodeFilter: &envoy_config_filter_accesslog_v2.StatusCodeFilter{
											Comparison: &envoy_config_filter_accesslog_v2.ComparisonFilter{
												Op: envoy_config_filter_accesslog_v2.ComparisonFilter_GE,
												Value: &envoy_api_v2_core.RuntimeUInt32{
													DefaultValue: uint32(200),
													RuntimeKey:   "test",
												},
											},
										},
									},
								}, {
									FilterSpecifier: &envoy_config_filter_accesslog_v2.AccessLogFilter_StatusCodeFilter{
										StatusCodeFilter: &envoy_config_filter_accesslog_v2.StatusCodeFilter{
											Comparison: &envoy_config_filter_accesslog_v2.ComparisonFilter{
												Op: envoy_config_filter_accesslog_v2.ComparisonFilter_EQ,
												Value: &envoy_api_v2_core.RuntimeUInt32{
													DefaultValue: uint32(0),
													RuntimeKey:   "test",
												},
											},
										},
									},
								}, {
									FilterSpecifier: &envoy_config_filter_accesslog_v2.AccessLogFilter_DurationFilter{
										DurationFilter: &envoy_config_filter_accesslog_v2.DurationFilter{
											Comparison: &envoy_config_filter_accesslog_v2.ComparisonFilter{
												Op: envoy_config_filter_accesslog_v2.ComparisonFilter_GE,
												Value: &envoy_api_v2_core.RuntimeUInt32{
													DefaultValue: uint32(2000),
													RuntimeKey:   "test",
												},
											},
										},
									},
								}},
							},
						},
					}, {
						FilterSpecifier: &envoy_config_filter_accesslog_v2.AccessLogFilter_NotHealthCheckFilter{
							NotHealthCheckFilter: &envoy_config_filter_accesslog_v2.NotHealthCheckFilter{},
						},
					}},
				},
			},
		},
		Config: s.accessLogConfig(fmt.Sprintf("%s_error.log", statPrefix)),
	}
}

func (s *lds) accessLog(statPrefix string) *envoy_config_filter_accesslog_v2.AccessLog {
	return &envoy_config_filter_accesslog_v2.AccessLog{
		Name: "envoy.file_access_log",
		Filter: &envoy_config_filter_accesslog_v2.AccessLogFilter{
			FilterSpecifier: &envoy_config_filter_accesslog_v2.AccessLogFilter_NotHealthCheckFilter{
				NotHealthCheckFilter: &envoy_config_filter_accesslog_v2.NotHealthCheckFilter{},
			},
		},
		Config: s.accessLogConfig(fmt.Sprintf("%s.log", statPrefix)),
	}
}

func (s *lds) manager() *types.Struct {
	logs := []*envoy_config_filter_accesslog_v2.AccessLog{}
	operation := hcm.EGRESS
	httpFilters := []*hcm.HttpFilter{{
		Name:   "envoy.router",
		Config: s.router(),
	}}

	statPrefix := ""

	if healthCheck := s.listener.Healthcheck; healthCheck != nil {
		healthCheckFilter := &hcm.HttpFilter{
			Name:   "envoy.health_check",
			Config: s.healthCheck(healthCheck),
		}
		httpFilters = append(httpFilters, healthCheckFilter)
	}

	if s.listener.Name == "local_service" {
		statPrefix = "ingress_http"
		operation = hcm.INGRESS

		logs = append(logs, s.accessLog(statPrefix))
		logs = append(logs, s.errorLog(statPrefix))
	} else {
		statPrefix = "egress_http"
		logs = append(logs, s.accessLog(statPrefix))
	}

	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: statPrefix,
		Tracing: &hcm.HttpConnectionManager_Tracing{
			OperationName: operation,
		},
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    s.configSource(),
				RouteConfigName: s.listener.Name,
			},
		},
		HttpFilters: httpFilters,
		AccessLog:   logs,
	}
	pbst, err := util.MessageToStruct(manager)
	if err != nil {
		panic(err)
	}
	return pbst
}

func (s *lds) configTLS() *envoy_api_v2_auth.DownstreamTlsContext {
	const base = "/etc/envoy/ssl/"
	return &envoy_api_v2_auth.DownstreamTlsContext{
		CommonTlsContext: &envoy_api_v2_auth.CommonTlsContext{
			TlsCertificates: []*envoy_api_v2_auth.TlsCertificate{{
				CertificateChain: &envoy_api_v2_core.DataSource{
					&envoy_api_v2_core.DataSource_Filename{
						Filename: filepath.Join(base, "envoy.crt"),
					},
				},
				PrivateKey: &envoy_api_v2_core.DataSource{
					&envoy_api_v2_core.DataSource_Filename{
						Filename: filepath.Join(base, "envoy.pem"),
					},
				},
			}},
			AlpnProtocols: []string{"h2", "http/1.1"},
		},
	}
}

func (s *lds) configFilterChain() envoy_api_v2_listener.FilterChain {
	var filterChain envoy_api_v2_listener.FilterChain
	if s.listener.TLS == true {
		filterChain = envoy_api_v2_listener.FilterChain{
			TlsContext: s.configTLS(),
			Filters: []envoy_api_v2_listener.Filter{{
				Name:   "envoy.http_connection_manager",
				Config: s.manager(),
			}},
		}
	} else {
		filterChain = envoy_api_v2_listener.FilterChain{
			Filters: []envoy_api_v2_listener.Filter{{
				Name:   "envoy.http_connection_manager",
				Config: s.manager(),
			}},
		}
	}
	return filterChain
}

func (s *lds) Listener() *envoy_api_v2.Listener {
	listener := &envoy_api_v2.Listener{
		Name: s.listener.Name,
		Address: envoy_api_v2_core.Address{
			Address: &envoy_api_v2_core.Address_SocketAddress{
				SocketAddress: &envoy_api_v2_core.SocketAddress{
					Protocol: envoy_api_v2_core.TCP,
					Address:  s.listener.Host,
					PortSpecifier: &envoy_api_v2_core.SocketAddress_PortValue{
						PortValue: uint32(s.listener.Port),
					},
				},
			},
		},
		FilterChains: []envoy_api_v2_listener.FilterChain{},
	}

	listener.FilterChains = append(listener.FilterChains, s.configFilterChain())
	return listener
}

func NewListener(listener ListenerConfig, ads bool) Listener {
	return &lds{listener: listener,
		ads: ads}
}
