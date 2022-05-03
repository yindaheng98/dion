package isglb

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/grpc"
)

const ServiceISGLB = "isglb"

type Config config.Common

func (c *Config) Load(file string) error {
	return config.Load(file, c)
}

type ISGLB struct {
	ion.Node
	s *ISGLBService
	runner.Service
	conf Config
	algC func() algorithms.Algorithm
}

// New create a new ISGLB
// algConstructor should return a algorithms.Algorithm
func New(algConstructor func() algorithms.Algorithm) *ISGLB {
	return &ISGLB{
		Node: ion.NewNode("isglb-" + util.RandomString(6)),
		algC: algConstructor,
	}
}

// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

func (s *ISGLB) ConfigBase() runner.ConfigBase {
	return &s.conf
}

// Load load config file
func (s *ISGLB) Load(confFile string) error {
	err := s.conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return err
	}
	return nil
}

// StartGRPC start with grpc.ServiceRegistrar
func (s *ISGLB) StartGRPC(registrar grpc.ServiceRegistrar) error {
	s.s = NewISGLBService(s.algC())
	pb.RegisterISGLBServer(registrar, s.s)
	log.Infof("sfu pb.RegisterISGLBServer(registrar, s.s)")
	return nil
}

func (s *ISGLB) Start(conf Config) error {
	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	s.s = NewISGLBService(s.algC())
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

// Close all
func (s *ISGLB) Close() {
	s.Node.Close()
}

// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
