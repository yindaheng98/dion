package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
	"os/exec"
)

// SimpleFFmpegTestsrcPublisher a Publisher get video from ffmpeg -f lavfi -i testsrc=XXX
// WARNING: 根本停不下来
type SimpleFFmpegTestsrcPublisher struct {
	PublisherFactory
	ffmpegPath string
	Filter     string
	Bandwidth  string
	Testsrc    string
}

func NewSimpleFFmpegTestsrcPublisher(ffmpegPath string, sfu *ion_sfu.SFU) SimpleFFmpegTestsrcPublisher {
	return SimpleFFmpegTestsrcPublisher{
		PublisherFactory: NewPublisherFactory(sfu),
		ffmpegPath:       ffmpegPath,
		Filter:           "drawbox=x=0:y=0:w=50:h=50:c=blue",
		Bandwidth:        "3M",
		Testsrc:          "size=1280x720:rate=30",
	}
}

func (p SimpleFFmpegTestsrcPublisher) NewDoor() (util.UnblockedDoor[SID], error) {
	pubDoor, err := p.PublisherFactory.NewDoor()
	if err != nil {
		log.Errorf("Cannot PublisherFactory.NewDoor: %+v", err)
		return nil, err
	}
	pub := pubDoor.(Publisher)
	err = p.makeTrack(pub)
	if err != nil {
		log.Errorf("Cannot makeTrack: %+v", err)
		return nil, err
	}
	return pub, nil
}

func (p SimpleFFmpegTestsrcPublisher) makeTrack(pub Publisher) error {
	videoTrack, err := MakeSampleIVFTrack(p.ffmpegPath, p.Testsrc, p.Filter, p.Bandwidth)
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

func MakeSampleIVFTrack(ffmpegPath, Testsrc, Filter, Bandwidth string) (webrtc.TrackLocal, error) {
	// Create a video track
	videoopt := []string{
		"-f", "lavfi",
		"-i", "testsrc=" + Testsrc,
		"-vf", Filter,
		"-vcodec", "libvpx",
		"-b:v", Bandwidth,
		"-f", "ivf",
		"pipe:1",
	}
	ffmpegCmd := exec.Command(ffmpegPath, videoopt...) //nolint
	_, ffmpegOut, err := util.GetStdPipes(ffmpegCmd)
	if err != nil {
		return nil, err
	}
	return util.MakeIVFTrackFromStdout(ffmpegOut, webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8})
}
