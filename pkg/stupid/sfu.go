package stupid

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"google.golang.org/grpc"
)

// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

// SFU represents a sfu node
type SFU struct {
	ion.Node
	s *sfu.SFUService
	runner.Service
	// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
	conf       sfu.Config
	ffmpegPath string
	Filter     string
	Bandwidth  string
	Testsrc    string
}

func (s *SFU) ConfigBase() runner.ConfigBase {
	return &s.conf
}

// New create a sfu node instance
func New(ffmpegPath string) *SFU {
	s := &SFU{
		Node:       ion.NewNode(config.ServiceNameStupid),
		ffmpegPath: ffmpegPath,
		Filter:     "drawbox=x=0:y=0:w=50:h=50:c=blue",
		Bandwidth:  "3M",
		Testsrc:    "size=1280x720:rate=30",
	}
	return s
}

// NewWithID create a sfu node instance with specific node id
func NewWithID(nid, ffmpegPath string) *SFU {
	s := &SFU{
		Node:       ion.NewNode(nid),
		ffmpegPath: ffmpegPath,
		Filter:     "drawbox=x=0:y=0:w=50:h=50:c=blue",
		Bandwidth:  "3M",
		Testsrc:    "size=1280x720:rate=30",
	}
	return s
}

// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

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
	// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
	isfu := ion_sfu.NewSFU(s.conf.Config)
	pub := bridge.NewSimpleFFmpegTestsrcPublisher(s.ffmpegPath, isfu)
	pub.Testsrc, pub.Filter, pub.Bandwidth = s.Testsrc, s.Filter, s.Bandwidth
	dog := util.NewWatchDogWithUnblockedDoor[bridge.SID](pub)
	dog.Watch(config.ServiceSessionStupid)
	s.s = sfu.NewSFUService(isfu)
	// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

	pb.RegisterRTCServer(registrar, s.s)
	log.Infof("sfu pb.RegisterRTCServer(registrar, s.s)")
	return nil
}

// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

// Start sfu node
func (s *SFU) Start(conf sfu.Config) error {
	// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
	isfu := ion_sfu.NewSFU(conf.Config)
	pub := bridge.NewSimpleFFmpegTestsrcPublisher(s.ffmpegPath, isfu)
	pub.Testsrc, pub.Filter, pub.Bandwidth = s.Testsrc, s.Filter, s.Bandwidth
	dog := util.NewWatchDogWithUnblockedDoor[bridge.SID](pub)
	dog.Watch(config.ServiceSessionStupid)
	s.s = sfu.NewSFUService(isfu)
	// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

	//grpc service
	pb.RegisterRTCServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: config.ServiceStupid,
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
