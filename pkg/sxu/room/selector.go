package room

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/yindaheng98/dion/config"
)

type Selector interface {
	Select(map[string]discovery.Node) []discovery.Node
}

type FirstSelector struct {
}

func (FirstSelector) Select(m map[string]discovery.Node) []discovery.Node {
	var nodes []discovery.Node
	for _, node := range m {
		if node.Service == config.ServiceSXU {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
