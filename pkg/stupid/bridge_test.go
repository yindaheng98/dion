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

	br := bridge.NewBridgeFactory(iSFU, bridge.NewSimpleFFmpegProcessor(ffmpeg))
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
