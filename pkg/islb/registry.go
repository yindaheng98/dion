package islb

import (
	"github.com/yindaheng98/dion/config"
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-discovery/pkg/registry"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
)

type Registry struct {
	dc    string
	reg   *registry.Registry
	mutex sync.RWMutex
	nodes map[string]discovery.Node
}

func NewRegistry(dc string, nc *nats.Conn) (*Registry, error) {

	reg, err := registry.NewRegistry(nc, config.DiscoveryExpire)
	if err != nil {
		log.Errorf("registry.NewRegistry: error => %v", err)
		return nil, err
	}

	r := &Registry{
		dc:    dc,
		reg:   reg,
		nodes: make(map[string]discovery.Node),
	}

	err = reg.Listen(r.handleNodeAction, r.handleGetNodes)

	if err != nil {
		log.Errorf("registry.Listen: error => %v", err)
		r.Close()
		return nil, err
	}

	return r, nil
}

func (r *Registry) Close() {
	r.reg.Close()
}

// handleNodeAction handle all Node from service discovery.
// This callback can observe all nodes in the ion cluster,
// TODO: Upload all node information to redis DB so that info
// can be shared when there are more than one ISLB in the later.
func (r *Registry) handleNodeAction(action discovery.Action, node discovery.Node) (bool, error) {
	//Add authentication here
	log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)

	//TODO: Put node info into the redis.
	r.mutex.Lock()
	defer r.mutex.Unlock()

	switch action {
	case discovery.Save:
		fallthrough
	case discovery.Update:
		r.nodes[node.ID()] = node
	case discovery.Delete:
		delete(r.nodes, node.ID())
	}

	return true, nil
}

func (r *Registry) handleGetNodes(service string, params map[string]interface{}) ([]discovery.Node, error) {
	//Add load balancing here.
	log.Infof("Get node by %v, params %v", service, params)

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var nodesResp []discovery.Node
	for _, item := range r.nodes {
		if item.Service == service || service == "*" {
			nodesResp = append(nodesResp, item)
		}
	}

	return nodesResp, nil
}
