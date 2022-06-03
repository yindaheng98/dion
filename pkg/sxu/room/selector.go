package room

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/islb"
)

type ClientFactory interface {
	NewClient() *rpc.Client
}

type FirstSelector struct {
	*islb.Node
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
