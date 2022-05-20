package bridge

import (
	"bufio"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/yindaheng98/dion/util"
	"io"
	"os/exec"
	"time"
)

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

func (p SimpleFFmpegTestsrcPublisher) NewDoor() (util.UnblockedDoor, error) {
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
	// Create a video track
	videoopt := []string{
		"-f", "lavfi",
		"-i", "testsrc=" + p.Testsrc,
		"-vf", p.Filter,
		"-vcodec", "libvpx",
		"-b:v", p.Bandwidth,
		"-f", "ivf",
		"pipe:1",
	}
	ffmpeg := exec.Command(p.ffmpegPath, videoopt...) //nolint
	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StdoutPipe(): %+v", err)
		return err
	}
	ffmpegErr, err := ffmpeg.StderrPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StderrPipe(): %+v", err)
		return err
	}

	if err := ffmpeg.Start(); err != nil {
		log.Errorf("Cannot Start ffmpeg: %+v", err)
		return err
	}

	go func(ffmpegErr io.ReadCloser) {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}(ffmpegErr)
	ivf, header, ivfErr := ivfreader.NewWith(ffmpegOut)
	if ivfErr != nil {
		log.Errorf("ivfreader create error: %+v", ivfErr)
		return ivfErr
	}

	videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if videoTrackErr != nil {
		log.Errorf("Cannot webrtc.NewTrackLocalStaticSample: %+v", err)
		return videoTrackErr
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

	go func() {
		// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
		// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
		//
		// It is important to use a time.Ticker instead of time.Sleep because
		// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
		// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
		ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000))
		for ; true; <-ticker.C {
			frame, _, ivfErr := ivf.ParseNextFrame()
			if ivfErr == io.EOF {
				fmt.Printf("All video frames parsed and sent")
			}
			if ivfErr != nil {
				panic(ivfErr)
			}
			if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
				log.Errorf("Cannot WriteSample: %+v", ivfErr)
			}
			fmt.Println("SimpleFFmpegTestsrcPublisher publish a RTP Packet")
		}
	}()

	return nil
}
