package sxu

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
	pbion "github.com/pion/ion/proto/ion"
	"github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/grpc"
	"testing"
	"time"
)

type SFU struct {
	ion.Node
	s *sfu.SFUService
	runner.Service
	conf Config

	sfu *ion_sfu.SFU

	toolbox ToolBoxBuilder
	syncer  *syncer.ISGLBSyncer
}

func NewSFU() *SFU {
	return &SFU{
		Node: ion.NewNode("sxu-test"),
	}
}

// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

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
	s.sfu = ion_sfu.NewSFU(s.conf.ISFU)
	s.s = sfu.NewSFUServiceWithSFU(s.sfu)
	rtc.RegisterRTCServer(registrar, s.s)
	log.Infof("sfu pb.RegisterRTCServer(registrar, s.s)")
	return nil
}

// Start sfu server
func (s *SFU) Start(conf Config) error {
	// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

	// Start internal SFU
	s.sfu = ion_sfu.NewSFU(conf.ISFU)

	// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓
	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	//s.s = sfu.NewSFUService(conf.Config)
	s.s = sfu.NewSFUServiceWithSFU(s.sfu)
	//grpc service
	rtc.RegisterRTCServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: ServiceSXU, //proto.ServiceRTC,
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

// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

func TestForwardTrackRoutineFactory(t *testing.T) {
	conf := Config{}
	err := conf.Load("D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml")
	if err != nil {
		panic(err)
	}

	sxu := NewSFU()
	err = sxu.Start(conf)
	if err != nil {
		t.Error(err)
		return
	}
	iSFU := sxu.sfu

	builder := DefaultToolBoxBuilder{}
	toolbox := builder.Build(&sxu.Node, iSFU)
	trackStupid := &pb.ForwardTrack{
		Src: &pbion.Node{
			Dc:      "dc1",
			Nid:     "stupid",
			Service: "rtc",
			Rpc:     nil,
		},
		RemoteSessionId: "stupid",
		LocalSessionId:  "stupid",
	}
	toolbox.TrackForwarder.StartForwardTrack(trackStupid)
	<-time.After(5 * time.Second)
	trackStupid2 := &pb.ForwardTrack{
		Src: &pbion.Node{
			Dc:      "dc1",
			Nid:     "stupid",
			Service: "rtc",
			Rpc:     nil,
		},
		RemoteSessionId: "stupid",
		LocalSessionId:  "stupid2",
	}
	toolbox.TrackForwarder.ReplaceForwardTrack(trackStupid, trackStupid2)
	<-time.After(5 * time.Second)
	toolbox.TrackForwarder.StartForwardTrack(trackStupid)
	<-time.After(5 * time.Second)
	toolbox.TrackForwarder.StopForwardTrack(trackStupid)
	<-time.After(5 * time.Second)
}
