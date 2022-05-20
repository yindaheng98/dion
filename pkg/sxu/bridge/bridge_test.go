package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

const YourName = "stupid2"

func TestEntrance(t *testing.T) {
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

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
