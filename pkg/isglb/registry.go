package isglb

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
)

func (isglb *ISGLBService) handleNodeAction(action discovery.Action, node discovery.Node) (bool, error) {
	//Add authentication here
	log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)
	switch action {
	case discovery.Delete:
		isglb.recvCh <- isglbRecvMessage{
			deleted: &node,
		}
	}

	return true, nil
}
