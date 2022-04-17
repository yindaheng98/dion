package isglb

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/isglb/algorithms"
	pb "github.com/yindaheng98/isglb/proto"
)

const ServiceISGLB = "isglb"

type global struct {
	Dc string `mapstructure:"dc"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

// Config defines parameters for the logger
type logConf struct {
	Level string `mapstructure:"level"`
}

// Config for sfu node
type Config struct {
	Global global   `mapstructure:"global"`
	Log    logConf  `mapstructure:"log"`
	Nats   natsConf `mapstructure:"nats"`
}

type ISGLB struct {
	ion.Node
	s *ISGLBService
}

func New(alg algorithms.Algorithm) *ISGLB {
	return &ISGLB{
		Node: ion.NewNode("isglb-" + util.RandomString(6)),
		s:    NewISGLBService(alg),
	}
}

func (s *ISGLB) Start(conf Config) error {
	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	//grpc service
	pb.RegisterISGLBServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: ServiceISGLB,
		NID:     s.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go func() {
		err := s.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("isglb.Node.KeepAlive(%v) error %v", s.Node.NID, err)
		}
	}()

	//Watch ALL nodes.
	go func() {
		err := s.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()

	return nil
}
