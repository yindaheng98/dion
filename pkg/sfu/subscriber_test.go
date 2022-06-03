package sfu

import (
	"fmt"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/pkg/islb"
	pb2 "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"testing"
	"time"
)

func TestSubscriber(t *testing.T) {
	node := islb.NewNode("sxu-" + util.RandomString(6))
	err := node.Start(conf.Nats.URL)
	if err != nil {
		t.Error(err)
	}
	//重要！！！必须开启了Watch才能自动地关闭NATS GRPC连接.
	go func() {
		err := node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()
	//重要！！！必须开启了KeepAlive才能在退出时让服务端那边自动地关闭NATS GRPC连接.
	go func() {
		err := node.KeepAlive(discovery.Node{
			DC:      conf.Global.Dc,
			Service: "test",
			NID:     node.NID,
			RPC: discovery.RPC{
				Protocol: discovery.NGRPC,
				Addr:     conf.Nats.URL,
				//Params:   map[string]string{"username": "foo", "password": "bar"},
			},
		})
		if err != nil {
			log.Errorf("isglb.Node.KeepAlive(%v) error %v", node.NID, err)
		}
	}()
	sub := NewSubscriber(&node)
	sub.OnTrack = func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
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
	}
	sub.SwitchSession(&pb2.ClientNeededSession{
		Session: "stupid",
		User:    "test",
	})
	<-time.After(10 * time.Second)
	sub.SwitchNode("unknown", map[string]interface{}{})
	<-time.After(10 * time.Second)
	sub.SwitchNode("*", map[string]interface{}{})
	<-time.After(10 * time.Second)
	sub.SwitchSession(&pb2.ClientNeededSession{
		Session: "unknown",
		User:    util.RandomString(8),
	})
	<-time.After(10 * time.Second)
	sub.SwitchSession(&pb2.ClientNeededSession{
		Session: "stupid",
		User:    util.RandomString(8),
	})
	<-time.After(10 * time.Second)
}
