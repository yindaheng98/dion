package stupid

import (
	"bufio"
	"fmt"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/sfu"
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
		"-vf", "drawbox=x=0:y=0:w=50:h=50:c=blue",
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

// makeVideo Make a video
func makeVideo(ffmpegPath, param, filter string) io.ReadCloser {
	videoopt := []string{
		"-f", "lavfi",
		"-i", "testsrc=" + param,
		"-vf", filter,
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

func (s *SFU) start(conf sfu.Config, iSFU *ion_sfu.SFU) error {
	// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

	err := s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	// ↑↑↑↑↑ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↑↑↑↑↑
	isfu := iSFU
	pub := NewPublisherFactory(s.in, isfu)
	dog := util.NewWatchDogWithUnblockedDoor(pub)
	dog.Watch(bridge.SID(config.ServiceSessionStupid))
	s.s = sfu.NewSFUServiceWithSFU(isfu)
	// ↓↓↓↓↓ Copy from https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/sfu.go ↓↓↓↓↓

	//grpc service
	rtc.RegisterRTCServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: config.ServiceStupid,
		NID:     s.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go func() {
		err := s.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("sfu.Node.KeepAlive(%v) error %v", s.Node.NID, err)
		}
	}()

	//Watch ALL nodes.
	go func() {
		err := s.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()

	return nil
}

func TestBridge(t *testing.T) {
	conf := sfu.Config{}
	file := "sfu.toml"
	ffmpeg := "D:\\Documents\\MyPrograms\\ffmpeg.exe"
	testvideo := "size=1280x720:rate=30"
	filter := "drawbox=x=w/2:y=h/2:w=50:h=50:c=red"

	err := conf.Load(file)
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		t.Error(err)
		return
	}

	fmt.Printf("config %s load ok!\n", file)

	log.Init(conf.Log.Level)

	log.Infof("--- making video ---")

	ffmpegOut := makeVideo(ffmpeg, testvideo, filter)

	log.Infof("--- starting bridge ---")

	iSFU := ion_sfu.NewSFU(conf.Config)

	br := bridge.NewBridgeFactory(iSFU, TestProcessor{ffmpegPath: ffmpeg})
	brDog := util.NewWatchDogWithUnblockedDoor(br)
	brDog.Watch(bridge.ProceedTrackParam{ProceedTrack: &pb.ProceedTrack{
		DstSessionId:     YourName,
		SrcSessionIdList: []string{config.ServiceSessionStupid},
	}})

	<-time.After(5 * time.Second)

	log.Infof("--- starting stupid node ---")

	server := New(ffmpegOut)
	if err := server.start(conf, iSFU); err != nil {
		panic(err)
	}
	defer server.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
