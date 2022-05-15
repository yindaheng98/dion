package bridge

import (
	"bufio"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

type TestProcessor struct {
	ffmpegPath string
}

func (t TestProcessor) AddTrack(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, OnBroken func(error)) (local webrtc.TrackLocal) {
	log.Warnf("onTrack: %+v", remote)
	videoopt := []string{
		"-f", "ivf",
		"-i", "pipe:0",
		"-vf", "drawtext=text='%{localtime\\:%Y-%M-%d %H.%m.%S}' :fontsize=240",
		"-vcodec", "libvpx",
		"-b:v", "3M",
		"-f", "ivf",
		"pipe:1",
	}
	ffmpeg := exec.Command(t.ffmpegPath, videoopt...) //nolint
	ffmpegIn, _ := ffmpeg.StdinPipe()
	ffmpegOut, _ := ffmpeg.StdoutPipe()
	ffmpegErr, _ := ffmpeg.StderrPipe()

	if err := ffmpeg.Start(); err != nil {
		panic(err)
	}

	// StdErr Scanner
	go func() {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video2", "pion2")
	if videoTrackErr != nil {
		panic(videoTrackErr)
	}

	// ffmpeg output reader
	go func() {
		ivf, header, ivfErr := ivfreader.NewWith(ffmpegOut)
		if ivfErr != nil {
			panic(ivfErr)
		}

		// Wait for connection established

		log.Warnf("onTrack ffmpeg out started: %+v", remote)

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
				OnBroken(fmt.Errorf("All video frames parsed and sent "))
				return
			}

			if ivfErr != nil {
				OnBroken(ivfErr)
				return
			}

			fmt.Println("RTP Packat write to SFU")
			if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
				OnBroken(ivfErr)
				return
			}
		}
	}()

	// ffmpeg in writer
	go func() {
		ivfWriter, err := ivfwriter.NewWith(ffmpegIn)
		if err != nil {
			panic(err)
		}
		fmt.Println("Track from SFU added")

		for {
			// Read RTP packets being sent to Pion
			rtp, _, readErr := remote.ReadRTP()
			fmt.Println("RTP Packat read from SFU")
			if readErr != nil {
				OnBroken(readErr)
				return
			}

			if ivfWriterErr := ivfWriter.WriteRTP(rtp); ivfWriterErr != nil {
				OnBroken(ivfWriterErr)
				return
			}
		}
	}()

	return videoTrack
}

func (t TestProcessor) UpdateProcedure(procedure *pb.ProceedTrack) {
	fmt.Printf("Updating: %+v\n", procedure)
}

const YourName = "stupid2"

func TestEntrance(t *testing.T) {
	confFile := "/root/Programs/dion/cmd/stupid/sfu.toml"
	ffmpeg := "/root/Programs/ffmpeg"
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
		road: TestProcessor{ffmpegPath: ffmpeg},
	}
	entdog := util.NewWatchDog(ent)
	entdog.Watch(SID(MyName))

	<-time.After(5 * time.Second)

	pub := NewTestPublisherFactory(ffmpegOut, iSFU)
	pubdog := util.NewWatchDog(pub)
	pubdog.Watch(SID(MyName))

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
