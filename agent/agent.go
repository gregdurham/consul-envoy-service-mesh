package agent

import (
	"github.com/hashicorp/consul/api"
)

type agent struct {
	host       string
	token      string
	datacenter string
}

//ConsulAgent describes consul agent behaviour
type ConsulAgent interface {
	CatalogServices() (map[string][]string, error)
	CatalogServiceEndpoints(serviceName string) ([]*api.CatalogService, error)
	WatchParams() map[string]string
	Keys(string, string) ([]string, error)
	ListKeys(string) (api.KVPairs, error)
}

//Catalog configures the consul agent catalog service client with the settings from current Agent.
func (a agent) catalog() (*api.Catalog, error) {
	config := api.DefaultConfig()
	config.Address = a.host
	config.Token = a.token
	config.Datacenter = a.datacenter
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client.Catalog(), nil
}

func (a agent) kv() (*api.KV, error) {
	config := api.DefaultConfig()
	config.Address = a.host
	config.Token = a.token
	config.Datacenter = a.datacenter
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return client.KV(), nil
}

func (a agent) CatalogServices() (map[string][]string, error) {
	catalog, err := a.catalog()
	if err != nil {
		return nil, err
	}
	services, _, err := catalog.Services(nil)
	return services, err
}

//CatalogServiceEndpoints makes an api call to consul agent host and gets list of catalog services
func (a agent) CatalogServiceEndpoints(serviceName string) ([]*api.CatalogService, error) {
	catalog, err := a.catalog()
	if err != nil {
		return nil, err
	}
	services, _, err := catalog.Service(serviceName, "", nil)
	return services, err
}

func (a agent) Keys(prefix string, sep string) ([]string, error) {
	kv, err := a.kv()
	pairs, _, err := kv.Keys(prefix, sep, nil)
	return pairs, err
}

func (a agent) ListKeys(prefix string) (api.KVPairs, error) {
	kv, err := a.kv()
	pairs, _, err := kv.List(prefix, nil)
	return pairs, err
}

func (a agent) WatchParams() map[string]string {
	return map[string]string{"datacenter": a.datacenter, "token": a.token}
}

//NewAgent creates a new instance of a ConsulAgent
func NewAgent(host, token, datacenter string) ConsulAgent {
	return agent{host: host, token: token, datacenter: datacenter}
}
