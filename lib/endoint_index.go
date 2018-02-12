package lib

import (
	"sync"
)

type EndpointIndex interface {
	Update(string, []*endpoint)
	GetEndpoints(string) ([]*endpoint, bool)
}

type endpointIndex struct {
	access    sync.Mutex
	endpoints map[string][]*endpoint
}

func (e *endpointIndex) Update(service string, endpoints []*endpoint) {
	e.access.Lock()
	e.endpoints[service] = endpoints
	e.access.Unlock()
}

func (e *endpointIndex) GetEndpoints(service string) ([]*endpoint, bool) {
	e.access.Lock()
	defer e.access.Unlock()
	if v, ok := e.endpoints[service]; ok {
		return v, ok
	} else {
		return nil, ok
	}
}

func NewEndpointIndex() EndpointIndex {
	return &endpointIndex{
		endpoints: make(map[string][]*endpoint),
	}
}
