package bridge

import (
	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/algorithms/impl/ffmpeg"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"testing"
	"time"
)

const YourName = "stupid2"

func TestBridge(t *testing.T) {
	confFile := "D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml"
	ffmpegPath := "D:\\Documents\\MyPrograms\\ffmpeg.exe"

	conf := readConf(confFile)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)
	dc := iSFU.NewDatachannel(ion_sfu.APIChannelLabel)
	dc.Use(datachannel.SubscriberAPI) // 没有初始化Datachannel会报错“SetRemoteDescription called with no ice-ufrag”，导致没有Track的时候无限制重启

	br := NewBridgeFactory(iSFU, ffmpeg.NewSimpleFFmpegIVFProcessorFactory(ffmpegPath))
	brdog := util.NewWatchDogWithUnblockedDoor[ProceedTrackParam](br)
	brdog.Watch(ProceedTrackParam{ProceedTrack: &pb.ProceedTrack{
		DstSessionId:     YourName,
		SrcSessionIdList: []string{MyName},
	}})

	<-time.After(5 * time.Second)

	pub := NewSimpleFFmpegTestsrcPublisher(ffmpegPath, iSFU)
	pubdog := util.NewWatchDogWithUnblockedDoor[SID](pub)
	pubdog.Watch(MyName)

	<-time.After(3 * time.Second)

	pub2 := NewSimpleFFmpegTestsrcPublisher(ffmpegPath, iSFU)
	pubdog2 := util.NewWatchDogWithUnblockedDoor[SID](pub2)
	pubdog2.Watch(MyName)

	<-time.After(1 * time.Second)

	pub3 := NewSimpleFFmpegTestsrcPublisher(ffmpegPath, iSFU)
	pubdog3 := util.NewWatchDogWithUnblockedDoor[SID](pub3)
	pubdog3.Watch(MyName)

	<-time.After(1 * time.Second)

	pubdog.Leave()
	pubdog2.Leave()

	<-time.After(5 * time.Second)

	brdog.Leave()

	<-time.After(2 * time.Second)
}
