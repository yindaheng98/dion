package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
)

const MyName = "stupid"

// makeVideo Make a video
func makeVideo(ffmpegPath, videosize string, framerate int) io.ReadCloser {
	videoopt := []string{
		"-f", "rawvideo",
		"-pixel_format", "yuv420p",
		"-video_size", videosize,
		"-framerate", strconv.Itoa(framerate),
		"-i", "/dev/urandom",
		"-vf", "drawtext=text='%{localtime\\:%Y-%M-%d %H.%m.%S}' :fontsize=120",
		"-vcodec", "libvpx",
		"-b:v", "3M",
		"-f", "ivf",
		"pipe:1",
	}
	ffmpeg := exec.Command(ffmpegPath, videoopt...) //nolint
	ffmpegOut, _ := ffmpeg.StdoutPipe()
	ffmpegErr, _ := ffmpeg.StderrPipe()

	if err := ffmpeg.Start(); err != nil {
		panic(err)
	}

	go func() {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()
	return ffmpegOut
}

// readConf Read a Config
func readConf(confFile string) Config {
	conf := Config{}
	err := conf.Load(confFile)
	if err != nil {
		panic(err)
	}
	return conf
}

// makePeer Make a Publisher
func makePub(iSFU *ion_sfu.SFU) bridge.Publisher {
	peer := ion_sfu.NewPeer(iSFU)

	// Make a Publisher
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}
	pub := bridge.NewPublisher(peer, pc)
	return pub
}

func makeTrack(ffmpegOut io.ReadCloser, pub bridge.Publisher) {
	// Create a video track
	videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if videoTrackErr != nil {
		panic(videoTrackErr)
	}

	rtpSender, videoTrackErr := pub.AddTrack(videoTrack)
	if videoTrackErr != nil {
		panic(videoTrackErr)
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

	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	go func() {
		ivf, header, ivfErr := ivfreader.NewWith(ffmpegOut)
		if ivfErr != nil {
			panic(ivfErr)
		}

		// Wait for connection established
		<-iceConnectedCtx.Done()

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
				os.Exit(0)
			}

			if ivfErr != nil {
				panic(ivfErr)
			}

			if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
				panic(ivfErr)
			}
		}
	}()

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	pub.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			iceConnectedCtxCancel()
		}
	})

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	pub.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}
	})
}

func main() {
	var confFile, session, ffmpeg, videosize string
	var framerate int
	flag.StringVar(&confFile, "conf", "cmd/stupid/sfu.toml", "sfu config file")
	flag.StringVar(&session, "session", MyName, "session of the video")
	flag.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "path to ffmpeg executable")
	flag.StringVar(&videosize, "videosize", "1280x720", "size of the video")
	flag.IntVar(&framerate, "framerate ", 30, "frame rate of the video")

	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}
	conf := readConf(confFile)

	ffmpegOut := makeVideo(ffmpeg, videosize, framerate)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)
	pub := makePub(iSFU)
	makeTrack(ffmpegOut, pub)
	err := pub.Publish(session)
	if err != nil {
		panic(err)
	}

	node := ion.NewNode(MyName)
	if err := node.Start(conf.Nats.URL); err != nil {
		panic(err)
	}
	defer node.Close()

	server := NewSFU()
	if err := server.Start(conf, iSFU); err != nil {
		panic(err)
	}
	defer server.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
