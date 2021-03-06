package sxu

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	ion_sfu_log "github.com/pion/ion-sfu/pkg/logger"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	pbion "github.com/pion/ion/proto/ion"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/islb"
	"github.com/yindaheng98/dion/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/grpc"
)

var logrLogger = ion_sfu_log.New().WithName("dion-sxu-node")

func init() {
	ion_sfu_log.SetGlobalOptions(ion_sfu_log.GlobalConfig{V: 1})
	ion_sfu.Logger = logrLogger.WithName("sxu")
}

type SXU struct {
	islb.Node
	ExtraInfo map[string]interface{} // 用来放置地域信息以供客户端进行选择

	s *sfu.SFUService
	runner.Service
	conf sfu.Config

	sfu *ion_sfu.SFU

	toolbox ToolBoxBuilder
	syncer  *syncer.ISGLBSyncer
}

func New(toolbox ToolBoxBuilder) *SXU {
	return NewWithID("sxu-"+util.RandomString(8), toolbox)
}

func NewWithID(id string, toolbox ToolBoxBuilder) *SXU {
	if toolbox == nil {
		toolbox = NewDefaultToolBoxBuilder()
	}
	return &SXU{
		Node:    islb.NewNode(id),
		toolbox: toolbox,
	}
}

// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

// Load load config file
func (s *SXU) Load(confFile string) error {
	err := s.conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return err
	}
	return nil
}

// StartGRPC start with grpc.ServiceRegistrar
func (s *SXU) StartGRPC(registrar grpc.ServiceRegistrar) error {
	//s.s = sfu.NewSFUService(s.conf.Config)
	// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

	// Start internal SFU
	s.sfu = ion_sfu.NewSFU(s.conf.Config)

	// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓
	s.s = sfu.NewSFUService(s.sfu)
	pb.RegisterRTCServer(registrar, s.s)
	log.Infof("sfu pb.RegisterRTCServer(registrar, s.s)")
	// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

	// Start syncer
	s.syncer = syncer.NewSFUStatusSyncer(&s.Node, "*", &pbion.Node{
		Dc:      s.conf.Global.Dc,
		Nid:     s.Node.NID,
		Service: config.ServiceSXU,
		Rpc:     nil,
	}, s.toolbox.Build(s, &s.Node, s.sfu))
	s.syncer.Start()

	// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓
	return nil
}

// Start sfu server
func (s *SXU) Start(conf sfu.Config) error {
	// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

	// Start internal SFU
	s.sfu = ion_sfu.NewSFU(conf.Config)

	// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓
	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	//s.s = sfu.NewSFUService(conf.Config)
	s.s = sfu.NewSFUService(s.sfu)
	//grpc service
	pb.RegisterRTCServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: config.ServiceSXU, //proto.ServiceRTC,
		NID:     s.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
		ExtraInfo: s.ExtraInfo,
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
	// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

	// Start syncer
	s.syncer = syncer.NewSFUStatusSyncer(&s.Node, "*", &pbion.Node{
		Dc:      conf.Global.Dc,
		Nid:     s.Node.NID,
		Service: config.ServiceSXU,
		Rpc:     nil,
	}, s.toolbox.Build(s, &s.Node, s.sfu))
	s.syncer.Start()

	// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓
	return nil
}

// Close all
func (s *SXU) Close() {
	s.syncer.Stop()
	s.Node.Close()
}

// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
