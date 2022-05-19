package stupid

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	"github.com/pion/ion/pkg/util"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/pkg/sfu"
	"google.golang.org/grpc"
)

type Config sfu.Config

// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

// SFU represents a sfu node
type SFU struct {
	ion.Node
	s *sfu.SFUService
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
func NewSFU(name string) *SFU {
	s := &SFU{
		Node: ion.NewNode(name),
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
	s.s = sfu.NewSFUService(s.conf.Config)
	pb.RegisterRTCServer(registrar, s.s)
	log.Infof("sfu pb.RegisterRTCServer(registrar, s.s)")
	return nil
}

// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

// Start sfu node
func (s *SFU) Start(conf Config, isfu *ion_sfu.SFU) error {
	// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
	s.s = sfu.NewSFUServiceWithSFU(isfu)
	// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

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
func (s *SFU) Close() {
	s.Node.Close()
}

// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
