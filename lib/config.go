package lib

import (
	"encoding/json"
	"fmt"

	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envoy_config_filter_http_health_check_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/health_check/v2"

	"github.com/gogo/protobuf/types"
)

type Configs []GenericConfig

type GenericConfig interface {
	config()
}

type ServiceConfig struct {
	ConfigType string   `json:"type"`
	Name       string   `json:"name"`
	Listeners  []string `json:"listeners"`
}

func (ServiceConfig) config() { return }

type ListenerConfig struct {
	ConfigType  string                                                `json:"type"`
	Name        string                                                `json:"name"`
	TLS         bool                                                  `json:"tls"`
	Host        string                                                `json:"host"`
	Port        int                                                   `json:"port"`
	Healthcheck *envoy_config_filter_http_health_check_v2.HealthCheck `json:"health_check,omitempty"`
	Clusters    []string                                              `json:"clusters"`
}

func (ListenerConfig) config() { return }

type ClusterConfig struct {
	ConfigType   string                                       `json:"type"`
	Name         string                                       `json:"name"`
	TLS          bool                                         `json:"tls"`
	Host         string                                       `json:"host,omitempty"`
	Port         int                                          `json:"port,omitempty"`
	Domains      []string                                     `json:"domains"`
	Hashpolicy   []*envoy_api_v2_route.RouteAction_HashPolicy `json:"hashpolicy,omitempty"`
	Prefix       string                                       `json:"prefix"`
	Protocol     string                                       `json:"protocol, omitempty"`
	Healthchecks []*envoy_api_v2_core.HealthCheck             `json:"health_checks,omitempty"`
}

func (ClusterConfig) config() { return }

type Healthcheck struct {
	HCType             string `json:"type"`
	Timeout            int    `json:"timeout_ms"`
	Interval           int    `json:"interval_ms"`
	UnhealthyThreshold int    `json:"unhealthy_threshold"`
	HealthyThreshold   int    `json:"healthy_threshold"`
	Host               string `json:"host"`
	Path               string `json:"path"`
	Send               string `json:"send"`
}

type Hashpolicy struct {
	HPType   string `json:"type"`
	Header   string `json:"header_name"`
	Cookie   string `json:"cookie_name"`
	TTL      int    `json:"ttl"`
	SourceIP bool   `json:"source_ip"`
}

func (g *Configs) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	var rawSingleton json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		if err := json.Unmarshal(b, &rawSingleton); err != nil {
			return err
		} else {
			raw = append(raw, rawSingleton)
		}
	}

	for _, r := range raw {
		var obj map[string]interface{}
		if err := json.Unmarshal(r, &obj); err != nil {
			fmt.Println(err)
			return err
		}
		configType := ""
		if t, ok := obj["type"].(string); ok {
			configType = t
		}

		var actual GenericConfig
		switch configType {
		case "cluster":
			actual = &ClusterConfig{}
		case "listener":
			actual = &ListenerConfig{}
		case "service":
			actual = &ServiceConfig{}
		}

		if err := json.Unmarshal(r, actual); err != nil {
			fmt.Println(err)
		}

		*g = append(*g, actual)
	}
	return nil
}

func (c *ClusterConfig) UnmarshalJSON(b []byte) error {
	type Alias ClusterConfig
	aux := &struct {
		Healthchecks []Healthcheck `json:"health_checks,omitempty"`
		Hashpolicy   []Hashpolicy  `json:"hash_policy,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	healthchecks := []*envoy_api_v2_core.HealthCheck{}
	for _, hc := range aux.Healthchecks {
		healthcheck := envoy_api_v2_core.HealthCheck{
			Timeout: &types.Duration{
				Nanos: int32(hc.Timeout * 1000000),
			},
			Interval: &types.Duration{
				Nanos: int32(hc.Timeout * 1000000),
			},
			UnhealthyThreshold: &types.UInt32Value{
				Value: uint32(hc.UnhealthyThreshold),
			},
			HealthyThreshold: &types.UInt32Value{
				Value: uint32(hc.HealthyThreshold),
			},
		}
		if hc.HCType == "http" {
			healthcheck.HealthChecker = &envoy_api_v2_core.HealthCheck_HttpHealthCheck_{
				HttpHealthCheck: &envoy_api_v2_core.HealthCheck_HttpHealthCheck{
					Path: hc.Path,
				},
			}
		}
		if hc.HCType == "tcp" {
			healthcheck.HealthChecker = &envoy_api_v2_core.HealthCheck_TcpHealthCheck_{
				TcpHealthCheck: &envoy_api_v2_core.HealthCheck_TcpHealthCheck{
					Send: &envoy_api_v2_core.HealthCheck_Payload{
						Payload: &envoy_api_v2_core.HealthCheck_Payload_Text{
							Text: hc.Send,
						},
					},
				},
			}
		}
		if hc.HCType == "redis" {
			healthcheck.HealthChecker = &envoy_api_v2_core.HealthCheck_RedisHealthCheck_{
				RedisHealthCheck: &envoy_api_v2_core.HealthCheck_RedisHealthCheck{},
			}
		}
		if hc.HCType == "grpc" {
			healthcheck.HealthChecker = &envoy_api_v2_core.HealthCheck_GrpcHealthCheck_{
				GrpcHealthCheck: &envoy_api_v2_core.HealthCheck_GrpcHealthCheck{},
			}
		}
		healthchecks = append(healthchecks, &healthcheck)
	}
	c.Healthchecks = healthchecks

	var hashpolicies []*envoy_api_v2_route.RouteAction_HashPolicy
	for _, hp := range aux.Hashpolicy {
		var hashpolicy envoy_api_v2_route.RouteAction_HashPolicy
		if hp.HPType == "cookie" {
			hashpolicy = envoy_api_v2_route.RouteAction_HashPolicy{
				PolicySpecifier: &envoy_api_v2_route.RouteAction_HashPolicy_Cookie_{
					Cookie: &envoy_api_v2_route.RouteAction_HashPolicy_Cookie{
						Name: hp.Cookie,
					},
				},
			}
		}

		if hp.HPType == "header" {
			hashpolicy = envoy_api_v2_route.RouteAction_HashPolicy{
				PolicySpecifier: &envoy_api_v2_route.RouteAction_HashPolicy_Header_{
					Header: &envoy_api_v2_route.RouteAction_HashPolicy_Header{
						HeaderName: hp.Header,
					},
				},
			}
		}

		if hp.HPType == "connection_properties" {
			hashpolicy = envoy_api_v2_route.RouteAction_HashPolicy{
				PolicySpecifier: &envoy_api_v2_route.RouteAction_HashPolicy_ConnectionProperties_{
					ConnectionProperties: &envoy_api_v2_route.RouteAction_HashPolicy_ConnectionProperties{
						SourceIp: hp.SourceIP,
					},
				},
			}
		}
		hashpolicies = append(hashpolicies, &hashpolicy)
	}
	c.Hashpolicy = hashpolicies

	return nil
}
