package sfu

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/node/sfu"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/config"
	"google.golang.org/grpc"
)

type Config config.Common

func (c *Config) Load(file string) error {
	return config.Load(file, c)
}

// SFUServer represents a sfu server
type SFUServer struct {
	*ion.Node
	s *sfu.SFUService
	runner.Service
	conf Config

	sfu *ion_sfu.SFU
}

func (s *SFUServer) ConfigBase() runner.ConfigBase {
	return &s.conf
}

// NewSFUServer create a sfu node instance
func NewSFUServer(node *ion.Node, sfu *ion_sfu.SFU) *SFUServer {
	s := &SFUServer{
		Node: node,
		sfu:  sfu,
	}
	return s
}

// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

// Load load config file
func (s *SFUServer) Load(confFile string) error {
	err := s.conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return err
	}
	return nil
}

// StartGRPC start with grpc.ServiceRegistrar
func (s *SFUServer) StartGRPC(registrar grpc.ServiceRegistrar) error {
	//s.s = sfu.NewSFUService(s.conf.Config)
	s.s = sfu.NewSFUServiceWithSFU(s.sfu)
	pb.RegisterRTCServer(registrar, s.s)
	log.Infof("sfu pb.RegisterRTCServer(registrar, s.s)")
	return nil
}

// Start sfu server
func (s *SFUServer) Start(conf Config) error {
	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	//s.s = sfu.NewSFUService(conf.Config)
	s.s = sfu.NewSFUServiceWithSFU(s.sfu)
	//grpc service
	pb.RegisterRTCServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: proto.ServiceRTC,
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
func (s *SFUServer) Close() {
	s.Node.Close()
}

// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
