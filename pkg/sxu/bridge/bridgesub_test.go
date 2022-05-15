package bridge

import (
	"context"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

type TestSubscriberFactory struct {
	SubscriberFactory
}

func NewTestSubscriberFactory(sfu *ion_sfu.SFU) TestSubscriberFactory {
	return TestSubscriberFactory{SubscriberFactory: SubscriberFactory{sfu: sfu}}
}

func (p TestSubscriberFactory) NewDoor() (util.Door, error) {
	subDoor, err := p.SubscriberFactory.NewDoor()
	if err != nil {
		log.Errorf("Cannot SubscriberFactory.NewDoor: %+v", err)
		return nil, err
	}
	sub := subDoor.(Subscriber)
	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())
	sub.SetOnConnectionStateChange(func(err error) {
		log.Errorf("onTrack closed: %+v", err)
	}, iceConnectedCtxCancel)
	sub.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Warnf("onTrack: %+v", remote)
		<-iceConnectedCtx.Done()
		log.Warnf("onTrack started: %+v", remote)

		for {
			// Read RTP packets being sent to Pion
			_, _, readErr := remote.ReadRTP()
			fmt.Println("RTP Packat read from SFU")
			if readErr != nil {
				panic(readErr)
			}
		}
	})
	return sub, nil
}

const MyName = "stupid"

func TestSubscriber(t *testing.T) {
	confFile := "/root/Programs/dion/cmd/stupid/sfu.toml"
	ffmpeg := "/root/Programs/ffmpeg"
	testvideo := "size=1280x720:rate=30"

	conf := readConf(confFile)

	ffmpegOut := makeVideo(ffmpeg, testvideo)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)

	sub := NewTestSubscriberFactory(iSFU)
	subdog := util.NewWatchDog(sub)
	subdog.Watch(SID(MyName))

	<-time.After(5 * time.Second)

	pub := NewTestPublisherFactory(ffmpegOut, iSFU)
	pubdog := util.NewWatchDog(pub)
	pubdog.Watch(SID(MyName))

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
