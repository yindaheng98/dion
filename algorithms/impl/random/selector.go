package random

import "github.com/cloudwebrtc/nats-discovery/pkg/discovery"

func RandomDiscoveryNode() discovery.Node {
	return discovery.Node{
		NID: "RandomDiscoveryNode-" + RandomString(8),
	}
}

type RandomSelector struct {
}

func (RandomSelector) Select(m map[string]discovery.Node) []discovery.Node {
	var nodes []discovery.Node
	for _, node := range m {
		if RandBool() {
			nodes = append(nodes, node)
		}
		if RandBool() {
			nodes = append(nodes, RandomDiscoveryNode())
		}
	}
	return nodes
}
