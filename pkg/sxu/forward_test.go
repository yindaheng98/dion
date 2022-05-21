package sxu

import (
	"fmt"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	pbion "github.com/pion/ion/proto/ion"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"testing"
	"time"
)

type SFU struct {
	ion.Node
	s *sfu.SFUService
	runner.Service
	conf sfu.Config

	sfu *ion_sfu.SFU

	toolbox ToolBoxBuilder
	syncer  *syncer.ISGLBSyncer
}

// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

// Start sfu server
func (s *SFU) Start(conf sfu.Config) error {
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
	s.s = sfu.NewSFUServiceWithSFU(s.sfu)
	//grpc service
	rtc.RegisterRTCServer(s.Node.ServiceRegistrar(), s.s)

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

// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑

type TestSubscriberFactory struct {
	bridge.SubscriberFactory
}

func NewTestSubscriberFactory(sfu *ion_sfu.SFU) TestSubscriberFactory {
	return TestSubscriberFactory{SubscriberFactory: bridge.NewSubscriberFactory(sfu)}
}

func (p TestSubscriberFactory) NewDoor() (util.UnblockedDoor, error) {
	subDoor, err := p.SubscriberFactory.NewDoor()
	if err != nil {
		log.Errorf("Cannot SubscriberFactory.NewDoor: %+v", err)
		return nil, err
	}
	sub := subDoor.(bridge.Subscriber)
	sub.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Warnf("onTrack started: %+v", remote)

		for {
			// Read RTP packets being sent to Pion
			_, _, readErr := remote.ReadRTP()
			fmt.Println("TestSubscriberFactory get a RTP Packet")
			if readErr != nil {
				fmt.Printf("TestSubscriberFactory RTP Packet read error %+v\n", readErr)
				return
			}
		}
	})
	return sub, nil
}

const MyName = "sxu-test"
const MySessionName = "stupid2"

func TestForwardTrackRoutineFactory(t *testing.T) {
	conf := sfu.Config{}
	err := conf.Load("D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml")
	if err != nil {
		panic(err)
	}

	sxu := &SFU{
		Node: ion.NewNode(MyName),
	}
	err = sxu.Start(conf)
	if err != nil {
		t.Error(err)
		return
	}
	iSFU := sxu.sfu

	sub := NewTestSubscriberFactory(iSFU)
	subdog := util.NewWatchDogWithUnblockedDoor(sub)
	subdog.Watch(bridge.SID(MySessionName))

	builder := NewDefaultToolBoxBuilder(WithSignallerFactory())
	toolbox := builder.Build(&sxu.Node, iSFU)

	trackStupid := &pb.ForwardTrack{
		Src: &pbion.Node{
			Dc:      "dc1",
			Nid:     config.ServiceNameStupid,
			Service: config.ServiceStupid,
			Rpc:     nil,
		},
		RemoteSessionId: config.ServiceSessionStupid,
		LocalSessionId:  MyName,
	}
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

	<-time.After(5 * time.Second)
	toolbox.TrackForwarder.StartForwardTrack(trackStupid)
	for {
		<-time.After(10 * time.Second)
		toolbox.TrackForwarder.ReplaceForwardTrack(trackStupid, trackStupid2)
		<-time.After(10 * time.Second)
		toolbox.TrackForwarder.ReplaceForwardTrack(trackStupid2, trackStupid)
	}
}
