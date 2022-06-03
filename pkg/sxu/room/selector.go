package room

import (
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/islb"
)

type ClientFactory interface {
	NewClient() *rpc.Client // when Client broken, this will be called
}

type FirstClientFactory struct {
	*islb.Node
}

func (f FirstClientFactory) NewClient() *rpc.Client {
	for _, node := range f.GetNeighborNodes() {
		if node.Service == config.ServiceSXU {
			client, err := f.NewNatsRPCClient(config.ServiceSXU, node.NID, map[string]interface{}{})
			if err != nil {
				log.Errorf("cannot NewNatsRPCClient: %v, try next", err)
			}
			return client
		}
	}
	return nil
}
