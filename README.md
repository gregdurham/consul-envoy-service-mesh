# Consul backed envoy service mesh
consul-envoy-service-mesh implements the envoy dataplane api, exposing configuration using xDS. This implementation exposes the following components:
- Listeners [LDS](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/dynamic_configuration.html#arch-overview-dynamic-config-lds)
- Endpoints [EDS](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/dynamic_configuration.html#sds-eds-only)
- Clusters [CDS](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/dynamic_configuration.html#sds-eds-only)
- Routes [RDS](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/dynamic_configuration.html#sds-eds-cds-and-rds)

Currently, this project exposes the [ADS](https://www.envoyproxy.io/docs/envoy/latest/configuration/overview/v2_overview.html#config-overview-v2-ads) (aggregated discovery service) which allows all of these components to be aggregated as a single configuration. 

## Configuration:

Requirements:
1) consul
2) consul-envoy-service-mesh binary
3) envoy

I will not go through the configuration of consul, you can get a container or download/install a binary, there are a number of good guides on this process available. 

However, you will now need to determine a kv path in consul which you will use for all envoy configurations. This path will be actively watched by `consul-envoy-service-mesh`. Any changes made in this path while the application is running will cause proxies to reconfigure. Once running, I would highly recommend changes made to this path happen via atomic operations, otherwise multi-key changes will cause multiple configuration changes to occur.

Sample configuration:
```
"envoy/test/"
"envoy/test/cluster/"
"envoy/test/cluster/local_service/"
"envoy/test/cluster/local_service/domains" => "*"
"envoy/test/cluster/local_service/host" => "127.0.0.1"
"envoy/test/cluster/local_service/listener" => "local_service"
"envoy/test/cluster/local_service/port" => 8080
"envoy/test/cluster/local_service/prefix" => "/"
"envoy/test/cluster/local_service/tls" => false
"envoy/test/cluster/rethinkdb/"
"envoy/test/cluster/rethinkdb/domains" => "*"
"envoy/test/cluster/rethinkdb/listener" => "internal_egress"
"envoy/test/cluster/rethinkdb/tls" => false
"envoy/test/listener/"
"envoy/test/listener/internal_egress/" 
"envoy/test/listener/internal_egress/host" => "127.0.0.1"
"envoy/test/listener/internal_egress/port" => 8182
"envoy/test/listener/internal_egress/tls" => false
"envoy/test/listener/local_service/"
"envoy/test/listener/local_service/host" => 0.0.0.0
"envoy/test/listener/local_service/port" => 8443
"envoy/test/listener/local_service/tls" => true
```

If TLS is true above, the certs are assumed to be installed in `/etc/envoy/conf.d/` the files are `cert.pem` and `priv.pem`. This will be configurable in the future.

Available options are here: 
- [cluster](https://github.com/gregdurham/consul-envoy-service-mesh/blob/master/config/cluster.go#L21)
- [listener](https://github.com/gregdurham/consul-envoy-service-mesh/blob/master/config/listener.go#L14)

The list of available options configurable will continue to grow. If you have requests, please create an issue, and they will be prioritized. Please submit PRs, it would be greatfully appreciated. 

In the near future, I will provide a container and possibly a binary of this application, however today you will need to build and install the binary. Once completed you can run it with the default options like so:
` ~/.go/bin/consul-envoy-service-mesh`

This will start the service on port `18000` 
Other options:

`~/.go/bin/consul-envoy-service-mesh --help`
```
Usage of /Users/gdurham/.go/bin/consul-envoy-service-mesh:
  -alsologtostderr
      log to standard error as well as files
  -configPath string
      consul kv path to configuration root (default "envoy/")
  -consulHost string
      consul hostname/ip (default "127.0.0.1")
  -consulPort uint
      consul port (default 8500)
  -http uint
      Http server port (default 18080)
  -log_backtrace_at value
      when logging hits line file:N, emit a stack trace
  -log_dir string
      If non-empty, write log files in this directory
  -logtostderr
      log to standard error instead of files
  -stderrthreshold value
      logs at or above this threshold go to stderr
  -v value
      log level for V logs
  -vmodule value
      comma-separated list of pattern=N settings for file-filtered logging
  -xds uint
      xDS server port (default 18000)
```

Initial sample **bootstrap** configuration:
```
node:
  id: test
  cluster: test

dynamic_resources:
  ads_config:
    api_type: GRPC
    cluster_name: [xds_cluster]
  lds_config: {ads: {}}
  cds_config: {ads: {}}

static_resources:
  clusters:
  - name: xds_cluster
    connect_timeout: 0.25s
    type: STATIC
    lb_policy: ROUND_ROBIN
    http2_protocol_options: {}
    hosts: [{ socket_address: { address: 192.168.100.1, port_value: 18000 }}]

admin:
  access_log_path: "/dev/null"
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 9901
```

This configuration tells envoy to connect to a host at address `192.168.100.1` for all resources on port `18000`. The node section is optionally configured via yaml or command line but MUST be set, see [here](https://www.envoyproxy.io/docs/envoy/latest/operations/cli.html?highlight=node#cmdoption-service-cluster) for options. 

# Things this was based on:
Some code and other ideas came from: [consul-envoy-xds](https://github.com/gojektech/consul-envoy-xds/)

I welcome any assistance, cleanup, or tips. Thank you, and hopefully this is useful to others. 



