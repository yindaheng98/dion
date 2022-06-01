package room

import "github.com/cloudwebrtc/nats-discovery/pkg/discovery"

type Selector interface {
	Select(map[string]discovery.Node) []discovery.Node
}
