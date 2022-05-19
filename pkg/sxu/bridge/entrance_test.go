package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
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
	ffmpeg := "D:\\Documents\\MyPrograms\\ffmpeg"
	testvideo := "size=1280x720:rate=30"

	conf := readConf(confFile)

	ffmpegOut := makeVideo(ffmpeg, testvideo)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)

	exitFact := NewPublisherFactory(iSFU)
	exitDoor, err := exitFact.NewDoor()
	if err != nil {
		panic(err)
	}
	exit := exitDoor.(Publisher)
	err = exit.Lock(SID(YourName), func(badGay error) {
		log.Errorf("bad gay comes: %+v", badGay)
		panic(badGay)
	})
	if err != nil {
		panic(err)
	}

	ent := EntranceFactory{
		SubscriberFactory: SubscriberFactory{
			sfu: iSFU,
		},
		exit: exit,
		road: NewSimpleFFmpegProcessor(ffmpeg),
	}
	entdog := util.NewWatchDogWithUnblockedDoor(ent)
	entdog.Watch(SID(MyName))

	<-time.After(5 * time.Second)

	pub := NewTestPublisherFactory(ffmpegOut, iSFU)
	pubdog := util.NewWatchDogWithUnblockedDoor(pub)
	pubdog.Watch(SID(MyName))

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
