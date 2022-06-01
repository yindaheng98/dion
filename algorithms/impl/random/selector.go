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
	nodes := make([]discovery.Node, len(m))
	i := 0
	for _, node := range m {
		nodes[i] = node
		if RandBool() {
			nodes[i] = RandomDiscoveryNode()
		}
		i++
	}
	return nodes
}
