package sxu

import (
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pbion "github.com/pion/ion/proto/ion"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"testing"
	"time"
)

type TestSubscriberFactory struct {
	bridge.SubscriberFactory
}

func NewTestSubscriberFactory(sfu *ion_sfu.SFU) TestSubscriberFactory {
	return TestSubscriberFactory{SubscriberFactory: bridge.NewSubscriberFactory(sfu)}
}

func (p TestSubscriberFactory) NewDoor() (util.UnblockedDoor[bridge.SID], error) {
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
const MySessionName = "sess-stupid"
const MySessionName2 = "sess-stupid2"

func TestForwardTrackRoutineFactory(t *testing.T) {
	conf := sfu.Config{}
	err := conf.Load("D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml")
	if err != nil {
		panic(err)
	}

	iSFU := ion_sfu.NewSFU(conf.Config)
	sxu := sfu.NewSFU(MyName)
	ss := sfu.NewSFUService(iSFU)
	err = sxu.Start(conf.Common, ss)
	if err != nil {
		t.Error(err)
		return
	}

	sub := NewTestSubscriberFactory(iSFU)
	subdog := util.NewWatchDogWithUnblockedDoor[bridge.SID](sub)
	subdog.Watch(MySessionName)

	sub2 := NewTestSubscriberFactory(iSFU)
	subdog2 := util.NewWatchDogWithUnblockedDoor[bridge.SID](sub2)
	subdog2.Watch(MySessionName2)

	builder := NewDefaultToolBoxBuilder(WithTrackForwarder())
	toolbox := builder.Build(&sxu.Node, iSFU)

	trackStupid := &pb.ForwardTrack{
		Src: &pbion.Node{
			Dc:      "dc1",
			Nid:     config.ServiceNameStupid,
			Service: config.ServiceStupid,
			Rpc:     nil,
		},
		RemoteSessionId: config.ServiceSessionStupid,
		LocalSessionId:  MySessionName,
	}
	trackStupid2 := &pb.ForwardTrack{
		Src: &pbion.Node{
			Dc:      "dc1",
			Nid:     config.ServiceNameStupid,
			Service: config.ServiceStupid,
			Rpc:     nil,
		},
		RemoteSessionId: config.ServiceSessionStupid,
		LocalSessionId:  MySessionName2,
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
