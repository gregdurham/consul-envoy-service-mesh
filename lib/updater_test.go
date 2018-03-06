package lib

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/cskr/pubsub"
	"github.com/envoyproxy/go-control-plane/pkg/cache"

	envoy_api_v2_core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

type Hasher struct {
}

func (h Hasher) ID(node *envoy_api_v2_core.Node) string {
	return node.GetId()
}

var ErrorQueue = make(chan error, 1000)

var (
	snapshot1   = make(map[string][]byte)
	snapshot2   = make(map[string][]byte)
	snapshot3   = make(map[string][]byte)
	snapshot4   = make(map[string][]byte)
	clusterJson = []byte(`{
		"type": "cluster",
		"name": "local_service",
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
	listenerJson = []byte(`{
		"type": "listener",
		"name": "test_listener",
		"tls": true,
		"host": "0.0.0.0",
		"port": 8443,
		"health_check":{
			"pass_through_mode": {"value": true},
			"endpoint": "/status"
		},
		"prefix": "/",
		"protocol": "http2",
		"clusters": [
			"local_service",
			"local_service_1"
		]
	}`)
)

func TestRun(t *testing.T) {
	snapshot1[cache.ListenerType] = []byte(`[{"name":"test_listener","address":{"Address":{"SocketAddress":{"address":"0.0.0.0","PortSpecifier":{"PortValue":8443}}}},"filter_chains":[{"tls_context":{"common_tls_context":{"tls_certificates":[{"certificate_chain":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.crt"}},"private_key":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.pem"}}}],"alpn_protocols":["h2","http/1.1"]},"SessionTicketKeysType":null},"filters":[{"name":"envoy.http_connection_manager","config":{"fields":{"access_log":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"format":{"Kind":{"StringValue":"[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\"\n"}},"path":{"Kind":{"StringValue":"/var/log/envoy/egress_http.log"}}}}}},"filter":{"Kind":{"StructValue":{"fields":{"not_health_check_filter":{"Kind":{"StructValue":{}}}}}}},"name":{"Kind":{"StringValue":"envoy.file_access_log"}}}}}}]}}},"http_filters":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{}}},"name":{"Kind":{"StringValue":"envoy.router"}}}}}},{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"endpoint":{"Kind":{"StringValue":"/status"}},"pass_through_mode":{"Kind":{"BoolValue":true}}}}}},"name":{"Kind":{"StringValue":"envoy.health_check"}}}}}}]}}},"rds":{"Kind":{"StructValue":{"fields":{"config_source":{"Kind":{"StructValue":{"fields":{"ads":{"Kind":{"StructValue":{}}}}}}},"route_config_name":{"Kind":{"StringValue":"test_listener"}}}}}},"stat_prefix":{"Kind":{"StringValue":"egress_http"}},"tracing":{"Kind":{"StructValue":{"fields":{"operation_name":{"Kind":{"StringValue":"EGRESS"}}}}}}}}}]}],"listener_filters":null}]`)
	snapshot1[cache.ClusterType] = []byte(`[{"name":"local_service","type":3,"eds_cluster_config":{"eds_config":{"ConfigSourceSpecifier":{"Ads":{}}},"service_name":"local_service"},"connect_timeout":5000000000,"health_checks":[{"timeout":{"nanos":1000000000},"interval":{"nanos":1000000000},"unhealthy_threshold":{"value":5},"healthy_threshold":{"value":6},"HealthChecker":{"HttpHealthCheck":{"path":"/status"}}}],"tls_context":{},"LbConfig":null},{"name":"local_service_1","type":3,"eds_cluster_config":{"eds_config":{"ConfigSourceSpecifier":{"Ads":{}}},"service_name":"local_service_1"},"connect_timeout":5000000000,"health_checks":[{"timeout":{"nanos":1000000000},"interval":{"nanos":1000000000},"unhealthy_threshold":{"value":5},"healthy_threshold":{"value":6},"HealthChecker":{"HttpHealthCheck":{"path":"/status"}}}],"tls_context":{},"LbConfig":null}]`)
	snapshot1[cache.EndpointType] = []byte(`[{"cluster_name":"local_service","endpoints":[{"locality":{"region":"dc1"},"lb_endpoints":[{"endpoint":{"address":{"Address":{"SocketAddress":{"address":"127.0.0.1","PortSpecifier":{"PortValue":8080}}}}},"health_status":1}]}],"policy":{}},{"cluster_name":"local_service_1","endpoints":[{"locality":{"region":"dc1"},"lb_endpoints":[{"endpoint":{"address":{"Address":{"SocketAddress":{"address":"127.0.0.1","PortSpecifier":{"PortValue":8080}}}}},"health_status":1}]}],"policy":{}}]`)
	snapshot1[cache.RouteType] = []byte(`[{"name":"test_listener","virtual_hosts":[{"name":"local_service","domains":["*"],"routes":[{"match":{"PathSpecifier":{"Prefix":"/"}},"Action":{"Route":{"ClusterSpecifier":{"Cluster":"local_service"},"HostRewriteSpecifier":null}},"decorator":{"operation":"local_service"}}]},{"name":"local_service_1","domains":["*"],"routes":[{"match":{"PathSpecifier":{"Prefix":"/"}},"Action":{"Route":{"ClusterSpecifier":{"Cluster":"local_service_1"},"HostRewriteSpecifier":null}},"decorator":{"operation":"local_service_1"}}]}]}]`)

	snapshot2[cache.ListenerType] = []byte(`[{"name":"test_listener","address":{"Address":{"SocketAddress":{"address":"0.0.0.0","PortSpecifier":{"PortValue":8443}}}},"filter_chains":[{"tls_context":{"common_tls_context":{"tls_certificates":[{"certificate_chain":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.crt"}},"private_key":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.pem"}}}],"alpn_protocols":["h2","http/1.1"]},"SessionTicketKeysType":null},"filters":[{"name":"envoy.http_connection_manager","config":{"fields":{"access_log":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"format":{"Kind":{"StringValue":"[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\"\n"}},"path":{"Kind":{"StringValue":"/var/log/envoy/egress_http.log"}}}}}},"filter":{"Kind":{"StructValue":{"fields":{"not_health_check_filter":{"Kind":{"StructValue":{}}}}}}},"name":{"Kind":{"StringValue":"envoy.file_access_log"}}}}}}]}}},"http_filters":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{}}},"name":{"Kind":{"StringValue":"envoy.router"}}}}}},{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"endpoint":{"Kind":{"StringValue":"/status"}},"pass_through_mode":{"Kind":{"BoolValue":true}}}}}},"name":{"Kind":{"StringValue":"envoy.health_check"}}}}}}]}}},"rds":{"Kind":{"StructValue":{"fields":{"config_source":{"Kind":{"StructValue":{"fields":{"ads":{"Kind":{"StructValue":{}}}}}}},"route_config_name":{"Kind":{"StringValue":"test_listener"}}}}}},"stat_prefix":{"Kind":{"StringValue":"egress_http"}},"tracing":{"Kind":{"StructValue":{"fields":{"operation_name":{"Kind":{"StringValue":"EGRESS"}}}}}}}}}]}],"listener_filters":null}]`)
	snapshot2[cache.ClusterType] = []byte(`[{"name":"local_service","type":3,"eds_cluster_config":{"eds_config":{"ConfigSourceSpecifier":{"Ads":{}}},"service_name":"local_service"},"connect_timeout":5000000000,"health_checks":[{"timeout":{"nanos":1000000000},"interval":{"nanos":1000000000},"unhealthy_threshold":{"value":5},"healthy_threshold":{"value":6},"HealthChecker":{"HttpHealthCheck":{"path":"/status"}}}],"tls_context":{},"LbConfig":null}]`)
	snapshot2[cache.EndpointType] = []byte(`[{"cluster_name":"local_service","endpoints":[{"locality":{"region":"dc1"},"lb_endpoints":[{"endpoint":{"address":{"Address":{"SocketAddress":{"address":"127.0.0.1","PortSpecifier":{"PortValue":5000}}}}},"health_status":1}]}],"policy":{}}]`)
	snapshot2[cache.RouteType] = []byte(`[{"name":"test_listener","virtual_hosts":[{"name":"local_service","domains":["*"],"routes":[{"match":{"PathSpecifier":{"Prefix":"/"}},"Action":{"Route":{"ClusterSpecifier":{"Cluster":"local_service"},"HostRewriteSpecifier":null}},"decorator":{"operation":"local_service"}}]}]}]`)

	snapshot3[cache.ListenerType] = []byte(`[{"name":"test_listener","address":{"Address":{"SocketAddress":{"address":"0.0.0.0","PortSpecifier":{"PortValue":8443}}}},"filter_chains":[{"tls_context":{"common_tls_context":{"tls_certificates":[{"certificate_chain":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.crt"}},"private_key":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.pem"}}}],"alpn_protocols":["h2","http/1.1"]},"SessionTicketKeysType":null},"filters":[{"name":"envoy.http_connection_manager","config":{"fields":{"access_log":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"format":{"Kind":{"StringValue":"[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\"\n"}},"path":{"Kind":{"StringValue":"/var/log/envoy/egress_http.log"}}}}}},"filter":{"Kind":{"StructValue":{"fields":{"not_health_check_filter":{"Kind":{"StructValue":{}}}}}}},"name":{"Kind":{"StringValue":"envoy.file_access_log"}}}}}}]}}},"http_filters":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{}}},"name":{"Kind":{"StringValue":"envoy.router"}}}}}},{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"endpoint":{"Kind":{"StringValue":"/status"}},"pass_through_mode":{"Kind":{"BoolValue":true}}}}}},"name":{"Kind":{"StringValue":"envoy.health_check"}}}}}}]}}},"rds":{"Kind":{"StructValue":{"fields":{"config_source":{"Kind":{"StructValue":{"fields":{"ads":{"Kind":{"StructValue":{}}}}}}},"route_config_name":{"Kind":{"StringValue":"test_listener"}}}}}},"stat_prefix":{"Kind":{"StringValue":"egress_http"}},"tracing":{"Kind":{"StructValue":{"fields":{"operation_name":{"Kind":{"StringValue":"EGRESS"}}}}}}}}}]}],"listener_filters":null}]`)
	snapshot3[cache.ClusterType] = []byte(`[{"name":"local_service","type":3,"eds_cluster_config":{"eds_config":{"ConfigSourceSpecifier":{"Ads":{}}},"service_name":"local_service"},"connect_timeout":5000000000,"health_checks":[{"timeout":{"nanos":1000000000},"interval":{"nanos":1000000000},"unhealthy_threshold":{"value":5},"healthy_threshold":{"value":6},"HealthChecker":{"HttpHealthCheck":{"path":"/status"}}}],"tls_context":{},"LbConfig":null}]`)
	snapshot3[cache.EndpointType] = []byte(`[null]`)
	snapshot3[cache.RouteType] = []byte(`[{"name":"test_listener","virtual_hosts":[{"name":"local_service","domains":["*"],"routes":[{"match":{"PathSpecifier":{"Prefix":"/"}},"Action":{"Route":{"ClusterSpecifier":{"Cluster":"local_service"},"HostRewriteSpecifier":null}},"decorator":{"operation":"local_service"}}]}]}]`)

	snapshot4[cache.ListenerType] = []byte(`[{"name":"test_listener","address":{"Address":{"SocketAddress":{"address":"0.0.0.0","PortSpecifier":{"PortValue":8443}}}},"filter_chains":[{"tls_context":{"common_tls_context":{"tls_certificates":[{"certificate_chain":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.crt"}},"private_key":{"Specifier":{"Filename":"/etc/envoy/ssl/envoy.pem"}}}],"alpn_protocols":["h2","http/1.1"]},"SessionTicketKeysType":null},"filters":[{"name":"envoy.http_connection_manager","config":{"fields":{"access_log":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"format":{"Kind":{"StringValue":"[%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\"\n"}},"path":{"Kind":{"StringValue":"/var/log/envoy/egress_http.log"}}}}}},"filter":{"Kind":{"StructValue":{"fields":{"not_health_check_filter":{"Kind":{"StructValue":{}}}}}}},"name":{"Kind":{"StringValue":"envoy.file_access_log"}}}}}}]}}},"http_filters":{"Kind":{"ListValue":{"values":[{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{}}},"name":{"Kind":{"StringValue":"envoy.router"}}}}}},{"Kind":{"StructValue":{"fields":{"config":{"Kind":{"StructValue":{"fields":{"endpoint":{"Kind":{"StringValue":"/status"}},"pass_through_mode":{"Kind":{"BoolValue":true}}}}}},"name":{"Kind":{"StringValue":"envoy.health_check"}}}}}}]}}},"rds":{"Kind":{"StructValue":{"fields":{"config_source":{"Kind":{"StructValue":{"fields":{"ads":{"Kind":{"StructValue":{}}}}}}},"route_config_name":{"Kind":{"StringValue":"test_listener"}}}}}},"stat_prefix":{"Kind":{"StringValue":"egress_http"}},"tracing":{"Kind":{"StructValue":{"fields":{"operation_name":{"Kind":{"StringValue":"EGRESS"}}}}}}}}}]}],"listener_filters":null}]`)
	snapshot4[cache.ClusterType] = []byte(`[{"name":"local_service","type":3,"eds_cluster_config":{"eds_config":{"ConfigSourceSpecifier":{"Ads":{}}},"service_name":"local_service"},"connect_timeout":5000000000,"health_checks":[{"timeout":{"nanos":1000000000},"interval":{"nanos":1000000000},"unhealthy_threshold":{"value":5},"healthy_threshold":{"value":6},"HealthChecker":{"HttpHealthCheck":{"path":"/status"}}}],"tls_context":{},"LbConfig":null}]`)
	snapshot4[cache.EndpointType] = []byte(`[{"cluster_name":"local_service","endpoints":[{"locality":{"region":"dc1"},"lb_endpoints":[{"endpoint":{"address":{"Address":{"SocketAddress":{"address":"192.168.1.1","PortSpecifier":{"PortValue":8500}}}}},"health_status":1}]}],"policy":{}}]`)
	snapshot4[cache.RouteType] = []byte(`[{"name":"test_listener","virtual_hosts":[{"name":"local_service","domains":["*"],"routes":[{"match":{"PathSpecifier":{"Prefix":"/"}},"Action":{"Route":{"ClusterSpecifier":{"Cluster":"local_service"},"HostRewriteSpecifier":null}},"decorator":{"operation":"local_service"}}]}]}]`)

	cluster, err := UnmarshalJson(clusterJson)
	if err != nil {
		t.Fail()
	}

	cluster2 := *cluster
	cluster2.Name = "local_service_1"

	listener, err := UnmarshalListenerConfigJson(listenerJson)
	if err != nil {
		t.Fail()
	}

	config := cache.NewSnapshotCache(true, Hasher{}, nil)
	resources := Resource{Clusters: []ClusterConfig{*cluster, cluster2}, Listeners: []ListenerConfig{*listener}}
	events := pubsub.New(1000)
	serviceEndpoints := NewEndpointIndex()
	node := &envoy_api_v2_core.Node{Id: "test"}

	watches := make(map[string]chan cache.Response)

	updater := NewUpdater("test", config, events, serviceEndpoints, ErrorQueue)
	updater.SetResources(resources)
	updater.Start()

	testCases := []struct {
		node     *envoy_api_v2_core.Node
		pre      func()
		snapshot map[string][]byte
		curVer   string
		nextVer  string
	}{
		{
			node:     node,
			snapshot: snapshot1,
			curVer:   "",
			nextVer:  "version0",
			pre:      func() {},
		},
		{
			node:     node,
			snapshot: snapshot2,
			curVer:   "version0",
			nextVer:  "version1",
			pre: func() {
				cluster.Port = 5000
				listener.Clusters = []string{"local_service"}
				resources = Resource{Clusters: []ClusterConfig{*cluster}, Listeners: []ListenerConfig{*listener}}
				updater.RefreshConfig(resources)
				time.Sleep(1)
			},
		},
		{
			node:     node,
			snapshot: snapshot3,
			curVer:   "version1",
			nextVer:  "version2",
			pre: func() {
				cluster.Port = 0
				cluster.Host = ""

				resources = Resource{Clusters: []ClusterConfig{*cluster}, Listeners: []ListenerConfig{*listener}}
				updater.RefreshConfig(resources)
				time.Sleep(1)
				if subs, err := updater.GetSubscriptions(); err != nil {
					t.Fail()
				} else if len(subs) != 1 {
					t.Fail()
				}
			},
		},
		{
			node:     node,
			snapshot: snapshot4,
			curVer:   "version2",
			nextVer:  "version3",
			pre: func() {
				serviceEndpoints.Update("local_service", []*endpoint{{address: "192.168.1.1", port: 8500}})
				rf := &watchOp{
					name:      "local_service",
					watchType: "service",
					operation: "refresh",
				}
				events.Pub(rf, "local_service")

			},
		},
	}
	for _, tc := range testCases {
		tc.pre()
		for _, typ := range cache.ResponseTypes {
			watches[typ], _ = config.CreateWatch(cache.Request{TypeUrl: typ, Node: tc.node, VersionInfo: tc.curVer})
		}
		for _, typ := range cache.ResponseTypes {
			t.Run(typ, func(t *testing.T) {
				select {
				case out := <-watches[typ]:
					if out.Version != tc.nextVer {
						t.Errorf("Got version %s, want %s", out.Version, tc.curVer)
					}
					if !reflect.DeepEqual(marshallResource(out.Resources), tc.snapshot[typ]) {
						t.Errorf("get resources %v, want %v", out.Resources, tc.snapshot[typ])
					}
				case <-time.After(time.Second):
					break
				}
			})
		}
	}

}

func marshallResource(resource []cache.Resource) []byte {
	out, err := json.Marshal(resource)
	if err != nil {
		return nil
	}
	return out

}
