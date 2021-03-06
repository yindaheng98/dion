package bridge

import (
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
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

func (p TestSubscriberFactory) NewDoor() (util.UnblockedDoor[SID], error) {
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
			fmt.Println("TestSubscriberFactory get a RTP Packet")
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
	dc := iSFU.NewDatachannel(ion_sfu.APIChannelLabel)
	dc.Use(datachannel.SubscriberAPI) // 没有初始化Datachannel会报错“SetRemoteDescription called with no ice-ufrag”，导致没有Track的时候无限制重启

	sub := NewTestSubscriberFactory(iSFU)
	subdog := util.NewWatchDogWithUnblockedDoor[SID](sub)
	subdog.Watch(MyName)

	<-time.After(5 * time.Second)

	pub := NewSimpleFFmpegTestsrcPublisher(ffmpeg, iSFU)
	pubdog := util.NewWatchDogWithUnblockedDoor[SID](pub)
	pubdog.Watch(MyName)

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
