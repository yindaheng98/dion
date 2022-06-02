package random

import (
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/util/ion"
)

type RandomClientFactory struct {
	*ion.Node
}

func (s RandomClientFactory) NewClient() *rpc.Client {
	for _, node := range s.GetNeighborNodes() {
		if RandBool() {
			client, err := s.NewNatsRPCClient(config.ServiceSXU, node.NID, map[string]interface{}{})
			if err != nil {
				log.Errorf("cannot NewNatsRPCClient: %v, try next", err)
			}
			return client
		}
	}
	log.Errorf("there is no nodes to be connect")
	return nil
}
