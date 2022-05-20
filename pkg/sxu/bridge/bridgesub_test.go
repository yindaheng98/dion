package bridge

import (
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

func (p TestSubscriberFactory) NewDoor() (util.UnblockedDoor, error) {
	subDoor, err := p.SubscriberFactory.NewDoor()
	if err != nil {
		log.Errorf("Cannot SubscriberFactory.NewDoor: %+v", err)
		return nil, err
	}
	sub := subDoor.(Subscriber)
	sub.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Warnf("onTrack started: %+v", remote)

		for {
			// Read RTP packets being sent to Pion
			_, _, readErr := remote.ReadRTP()
			fmt.Println("TestSubscriberFactory get a RTP Packat")
			if readErr != nil {
				panic(readErr)
			}
		}
	})
	return sub, nil
}

const MyName = "stupid"

func TestSubscriber(t *testing.T) {
	confFile := "D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml"
	ffmpeg := "D:\\Documents\\MyPrograms\\ffmpeg.exe"

	conf := readConf(confFile)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)

	sub := NewTestSubscriberFactory(iSFU)
	subdog := util.NewWatchDogWithUnblockedDoor(sub)
	subdog.Watch(SID(MyName))

	<-time.After(5 * time.Second)

	pub := NewSimpleFFmpegTestsrcPublisher(ffmpeg, iSFU)
	pubdog := util.NewWatchDogWithUnblockedDoor(pub)
	pubdog.Watch(SID(MyName))

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
