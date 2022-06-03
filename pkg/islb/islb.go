package islb

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/dion/config"
)

// ISLB represents islb node
type ISLB struct {
	Node
	registry *Registry
}

// NewISLB create a islb node instance
func NewISLB() *ISLB {
	return &ISLB{Node: NewNode("islb-" + util.RandomString(6))}
}

// Start islb node
func (i *ISLB) Start(conf config.Common) error {
	var err error

	err = i.Node.Start(conf.Nats.URL)
	if err != nil {
		i.Close()
		return err
	}

	//registry for node discovery.
	i.registry, err = NewRegistry(conf.Global.Dc, i.Node.NatsConn())
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	// Register reflection service on nats-rpc server.
	reflection.Register(i.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: proto.ServiceISLB,
		NID:     i.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
		},
	}

	go func() {
		err := i.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("islb.Node.KeepAlive: error => %v", err)
		}
	}()

	//Watch ALL nodes.
	go func() {
		err := i.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()

	return nil
}

// Close all
func (i *ISLB) Close() {
	i.Node.Close()
	if i.registry != nil {
		i.registry.Close()
	}
}
