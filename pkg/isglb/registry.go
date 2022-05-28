package isglb

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/config"
)

func (isglb *ISGLBService) handleNodeAction(action discovery.Action, node discovery.Node) (bool, error) {
	//Add authentication here
	log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)
	if action == discovery.Delete && node.Service == config.ServiceSXU {
		isglb.recvCh <- isglbRecvMessage{
			deleted: &node,
		}
	}
	// TODO: 当节点下线的时候自动删除之
	return true, nil
}
