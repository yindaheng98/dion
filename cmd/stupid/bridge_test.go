package main

import (
	"bufio"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
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

const YourName = "stupid2"

type TestProcessor struct {
	ffmpegPath string
}

func (t TestProcessor) AddTrack(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, OnBroken func(error)) (local webrtc.TrackLocal) {
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
		OnBroken(err)
		return
	}

	go func() {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video2", "pion2")
	if videoTrackErr != nil {
		OnBroken(videoTrackErr)
		return
	}

	go func() {
		ivf, header, ivfErr := ivfreader.NewWith(ffmpegOut)
		if ivfErr != nil {
			OnBroken(ivfErr)
			return
		}

		ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000))
		for ; true; <-ticker.C {
			frame, _, ivfErr := ivf.ParseNextFrame()
			if ivfErr == io.EOF {
				fmt.Printf("All video frames parsed and sent")
				OnBroken(ivfErr)
				return
			}

			if ivfErr != nil {
				OnBroken(ivfErr)
				return
			}

			if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
				OnBroken(ivfErr)
				return
			}
		}
	}()

	go func() {
		ivfWriter, err := ivfwriter.NewWith(ffmpegIn)
		if err != nil {
			OnBroken(err)
			return
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

func TestBridge(t *testing.T) {
	confFile := "/root/Programs/dion/cmd/stupid/sfu.toml"
	ffmpeg := "/root/Programs/ffmpeg"
	testvideo := "size=1280x720:rate=30"
	filter := "drawtext=text='%{localtime\\:%Y-%M-%d %H.%m.%S}':fontsize=60:x=(w-text_w)/2:y=(h-text_h)/2"

	conf := readConf(confFile)

	ffmpegOut := makeVideo(ffmpeg, testvideo, filter)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)

	br := bridge.NewBridgeFactory(iSFU, TestProcessor{ffmpegPath: ffmpeg})
	brDog := util.NewWatchDogWithUnblockedDoor(br)
	brDog.Watch(bridge.ProceedTrackParam{ProceedTrack: &pb.ProceedTrack{
		DstSessionId:     YourName,
		SrcSessionIdList: []string{MyName},
	}})

	<-time.After(5 * time.Second)

	pub := NewPublisherFactory(ffmpegOut, iSFU)
	dog := util.NewWatchDogWithUnblockedDoor(pub)
	dog.Watch(bridge.SID(MyName))

	node := ion.NewNode(MyName)
	if err := node.Start(conf.Nats.URL); err != nil {
		panic(err)
	}
	defer node.Close()

	server := NewSFU(MyName)
	if err := server.Start(conf, iSFU); err != nil {
		panic(err)
	}
	defer server.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
