package lib

import (
	"encoding/json"
	"errors"
	"testing"

	envoy_api_v2_auth "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/stretchr/testify/assert"
)

func UnmarshalListenerConfigJson(data []byte) (*ListenerConfig, error) {
	var v Configs
	var l *ListenerConfig
	if err := json.Unmarshal(data, &v); err != nil {
		return l, err
	}

	for _, config := range v {
		if listener, ok := config.(*ListenerConfig); ok {
			return listener, nil
		} else {
			return l, errors.New("Not a cluster")
		}
	}
	return l, errors.New("Not a cluster")
}

func TestShouldHaveBasicListenerSettings(t *testing.T) {
	data := []byte(`{
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
			"test_cluster"
		]
	}`)
	listener, err := UnmarshalListenerConfigJson(data)
	if err != nil {
		t.Fail()
	}
	envoyListener := NewListener(*listener, true).Listener()
	assert.Equal(t, "test_listener", envoyListener.Name)
	assert.Equal(t, "socket_address:<address:\"0.0.0.0\" port_value:8443 > ", envoyListener.Address.String())
}

func TestShouldHaveBasicListenerSettingsWithHttp2AndTLS(t *testing.T) {
	data := []byte(`{
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
			"test_cluster"
		]
	}`)
	listener, err := UnmarshalListenerConfigJson(data)
	if err != nil {
		t.Fail()
	}
	envoyListener := NewListener(*listener, true).Listener()
	assert.Equal(t, []string{"h2", "http/1.1"}, envoyListener.FilterChains[0].TlsContext.CommonTlsContext.AlpnProtocols)
	certs := envoyListener.FilterChains[0].TlsContext.CommonTlsContext.GetTlsCertificates()[0]
	assert.Equal(t, "filename:\"/etc/envoy/ssl/envoy.crt\" ", certs.CertificateChain.String())
	assert.Equal(t, "filename:\"/etc/envoy/ssl/envoy.pem\" ", certs.PrivateKey.String())
}

func TestShouldHaveBasicListenerSettingsWithOutTLS(t *testing.T) {
	data := []byte(`{
		"type": "listener",
		"name": "test_listener",
		"tls": false,
		"host": "0.0.0.0",
		"port": 8443,
		"health_check":{
			"pass_through_mode": {"value": true},
			"endpoint": "/status"
		},
		"prefix": "/",
		"protocol": "http2",
		"clusters": [
			"test_cluster"
		]
	}`)
	listener, err := UnmarshalListenerConfigJson(data)
	if err != nil {
		t.Fail()
	}
	envoyListener := NewListener(*listener, true).Listener()
	var tlsContext *envoy_api_v2_auth.DownstreamTlsContext
	assert.Equal(t, tlsContext, envoyListener.FilterChains[0].TlsContext)
}
