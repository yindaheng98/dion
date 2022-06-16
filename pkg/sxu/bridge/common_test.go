package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sfu"
	"github.com/yindaheng98/dion/util"
)

// readConf Read a Config
func readConf(confFile string) sfu.Config {
	conf := sfu.Config{}
	err := conf.Load(confFile)
	if err != nil {
		panic(err)
	}
	return conf
}

// SimpleFFmpegTestsrcPublisher a Publisher get video from ffmpeg -f lavfi -i testsrc=XXX
// WARNING: 根本停不下来
type SimpleFFmpegTestsrcPublisher struct {
	PublisherFactory
	ffmpegPath string
}

func NewSimpleFFmpegTestsrcPublisher(ffmpegPath string, sfu *ion_sfu.SFU) SimpleFFmpegTestsrcPublisher {
	return SimpleFFmpegTestsrcPublisher{
		PublisherFactory: NewPublisherFactory(sfu),
		ffmpegPath:       ffmpegPath,
	}
}

func (p SimpleFFmpegTestsrcPublisher) NewDoor() (util.UnblockedDoor[SID], error) {
	pubDoor, err := p.PublisherFactory.NewDoor()
	if err != nil {
		log.Errorf("Cannot PublisherFactory.NewDoor: %+v", err)
		return nil, err
	}
	pub := pubDoor.(Publisher)
	err = makeTrack(p.ffmpegPath, pub)
	if err != nil {
		log.Errorf("Cannot makeTrack: %+v", err)
		return nil, err
	}
	return pub, nil
}

func makeTrack(ffmpegPath string, pub Publisher) error {
	videoTrack, err := util.MakeSampleIVFTrack(ffmpegPath,
		"size=1280x720:rate=30",
		"drawbox=x=0:y=0:w=50:h=50:c=blue",
		"3M")
	if err != nil {
		return err
	}

	rtpSender, videoTrackErr := pub.AddTrack(videoTrack)
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
