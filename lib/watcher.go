package lib

import (
	"github.com/cskr/pubsub"
	"github.com/golang/glog"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
)

type Watcher struct {
	ID               string
	WatchType        string
	Plan             *watch.Plan
	AgentHost        string
	Events           *pubsub.PubSub
	ServiceEndpoints EndpointIndex
	QuitChan         chan bool
	ErrorChan        chan error
}

func NewWatcher(watchType string, id string, agentHost string, serviceEndpoints EndpointIndex, events *pubsub.PubSub, errorChan chan error) Watcher {
	watcher := Watcher{
		WatchType:        watchType,
		ID:               id,
		AgentHost:        agentHost,
		Events:           events,
		ServiceEndpoints: serviceEndpoints,
		QuitChan:         make(chan bool),
		ErrorChan:        errorChan,
	}

	return watcher
}

type watchOp struct {
	name, watchType, operation string
}

func (w *Watcher) Start() {
	go func() {
		for {
			glog.V(10).Infof("Starting %s", w.ID)
			if err := w.Plan.Run(w.AgentHost); err != nil {
				glog.V(10).Infof("Error when attempting to execute the plan: %s", err)
				w.ErrorChan <- err
			}

			select {
			case <-w.QuitChan:
				glog.V(10).Infof("Received request to quit %s", w.ID)
				return
			}
		}
	}()
}

func (w *Watcher) WatchPlan() {
	if w.WatchType == "keyprefix" {
		w.KeyPrefixWatchPlan()
	} else if w.WatchType == "service" {
		w.ServiceWatchPlan()
	}
}

func (w *Watcher) ServiceWatchPlan() {

	plan, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": w.ID,
	})

	if err != nil {
		w.ErrorChan <- err
	}

	plan.Handler = func(idx uint64, data interface{}) {
		w.ServiceWatchHandler(data)
	}

	w.Plan = plan

}

func (w *Watcher) KeyPrefixWatchPlan() {

	plan, err := watch.Parse(map[string]interface{}{
		"type":   "keyprefix",
		"prefix": w.ID,
	})

	if err != nil {
		w.ErrorChan <- err
	}

	plan.Handler = func(idx uint64, data interface{}) {
		w.KeyPrefixWatchHandler(data)
	}

	w.Plan = plan

}

func (w *Watcher) ServiceWatchHandler(data interface{}) {
	v, ok := data.([]*api.ServiceEntry)
	if ok {
		endpoints := make([]*endpoint, 0)
		for _, e := range v {
			ep := &endpoint{
				address: e.Service.Address,
				port:    e.Service.Port,
			}
			endpoints = append(endpoints, ep)
		}
		w.ServiceEndpoints.Update(w.ID, endpoints)
		rf := &watchOp{
			name:      w.ID,
			watchType: "service",
			operation: "refresh",
		}
		w.Events.Pub(rf, w.ID)
	}
}

func (w *Watcher) KeyPrefixWatchHandler(data interface{}) {
	_, ok := data.(api.KVPairs)
	if ok {
		rf := &watchOp{
			name:      w.ID,
			watchType: "keyprefix",
			operation: "refresh",
		}
		w.Events.Pub(rf, w.ID)
	}
}

func (w *Watcher) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}
