package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"testing"
	"time"
)

const YourName = "stupid2"

func TestBridge(t *testing.T) {
	confFile := "D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml"
	ffmpeg := "D:\\Documents\\MyPrograms\\ffmpeg.exe"

	conf := readConf(confFile)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)

	br := NewBridgeFactory(iSFU, NewSimpleFFmpegIVFProcessorFactory(ffmpeg))
	brdog := util.NewWatchDogWithUnblockedDoor(br)
	brdog.Watch(ProceedTrackParam{ProceedTrack: &pb.ProceedTrack{
		DstSessionId:     YourName,
		SrcSessionIdList: []string{MyName},
	}})

	<-time.After(5 * time.Second)

	pub := NewSimpleFFmpegTestsrcPublisher(ffmpeg, iSFU)
	pubdog := util.NewWatchDogWithUnblockedDoor(pub)
	pubdog.Watch(SID(MyName))

	<-time.After(3 * time.Second)

	pub2 := NewSimpleFFmpegTestsrcPublisher(ffmpeg, iSFU)
	pubdog2 := util.NewWatchDogWithUnblockedDoor(pub2)
	pubdog2.Watch(SID(MyName))

	<-time.After(1 * time.Second)

	pub3 := NewSimpleFFmpegTestsrcPublisher(ffmpeg, iSFU)
	pubdog3 := util.NewWatchDogWithUnblockedDoor(pub3)
	pubdog3.Watch(SID(MyName))

	<-time.After(1 * time.Second)

	pubdog.Leave()
	pubdog2.Leave()

	<-time.After(5 * time.Second)

	brdog.Leave()

	<-time.After(2 * time.Second)
}
