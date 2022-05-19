package sfu

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	"github.com/pion/ion/pkg/util"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/config"
	"google.golang.org/grpc"
)

// SFU represents a sfu node
type SFU struct {
	ion.Node
	s *SFUService
	runner.Service
	conf Config
}

// New create a sfu node instance
func New() *SFU {
	s := &SFU{
		Node: ion.NewNode("sfu-" + util.RandomString(6)),
	}
	return s
}

func (s *SFU) ConfigBase() runner.ConfigBase {
	return &s.conf
}

// NewSFU create a sfu node instance
func NewSFU(id string) *SFU {
	s := &SFU{
		Node: ion.NewNode(id),
	}
	return s
}

// Load load config file
func (s *SFU) Load(confFile string) error {
	err := s.conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return err
	}
	return nil
}

// StartGRPC start with grpc.ServiceRegistrar
func (s *SFU) StartGRPC(registrar grpc.ServiceRegistrar) error {
	s.s = NewSFUService(s.conf.Config)
	pb.RegisterRTCServer(registrar, s.s)
	log.Infof("sfu pb.RegisterRTCServer(registrar, s.s)")
	return nil
}

// Start sfu node
func (s *SFU) Start(conf Config) error {
	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	s.s = NewSFUService(conf.Config)
	//grpc service
	pb.RegisterRTCServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: config.ServiceSXU,
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
			log.Errorf("sfu.Node.KeepAlive(%v) error %v", s.Node.NID, err)
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

// Close all
func (s *SFU) Close() {
	s.Node.Close()
}
