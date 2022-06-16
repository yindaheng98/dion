package sfu

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/islb"
	pb2 "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"testing"
)

func TestPublisher(t *testing.T) {
	ffmpegPath := "D:\\Documents\\MyPrograms\\ffmpeg.exe"

	node := islb.NewNode("cli-" + util.RandomString(6))
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
			Service: config.ServiceClient,
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
	pub := NewPublisher(&node)
	videoTrack, err := util.MakeSampleIVFTrack(
		ffmpegPath,
		"size=1280x720:rate=30",
		"drawbox=x=0:y=0:w=50:h=50:c=blue",
		"3M",
	)
	if err != nil {
		panic(err)
	}
	pub.NeedTrack = func(AddTrack func(track webrtc.TrackLocal) (*webrtc.RTPSender, error)) error {
		log.Warnf("needTrack started")

		rtpSender, videoTrackErr := AddTrack(videoTrack)
		if videoTrackErr != nil {
			return videoTrackErr
		}
		// Read incoming RTCP packets
		// Before these packets are returned they are processed by interceptors. For things
		// like NACK this needs to be called.
		go func() {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
					return
				}
			}
		}()

		return nil
	}
	pub.Switch(config.ServiceNameStupid, map[string]interface{}{}, &pb2.ClientNeededSession{
		Session: "test",
		User:    "publisher",
	})
	select {}
}
