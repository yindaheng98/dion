package main

import (
	"flag"
	"fmt"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/islb"
	"github.com/yindaheng98/dion/pkg/sfu"
	pb2 "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var (
	conf = config.Common{}
	file string
)

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
	os.Exit(-1)
}

func main() {
	var ffmpeg, device, nid, sid, uid string
	flag.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "path to ffmpeg executable")
	flag.StringVar(&device, "device",
		"@device_pnp_\\\\?\\usb#vid_2bdf&pid_028a&mi_00#6&1d424522&0&0000#{65e8773d-8f56-11d0-a3b9-00a0c9223196}\\global",
		"device id of your camera (use 'ffmpeg -list_devices true -f dshow -i dummy' to show it)")
	flag.StringVar(&nid, "nid", "stupid", "target node id")
	flag.StringVar(&sid, "sid", "camera", "target session id")
	flag.StringVar(&uid, "uid", util.RandomString(8), "your user id")
	flag.StringVar(&file, "c", "cmd/islb/islb.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if *help {
		showHelp()
	}

	err := conf.Load(file)
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		showHelp()
	}

	fmt.Printf("config %s load ok!\n", file)

	log.Init(conf.Log.Level)

	node := islb.NewNode("sxu-" + util.RandomString(6))

	err = node.Start(conf.Nats.URL)
	if err != nil {
		panic(err)
	}
	//重要！！！必须开启了Watch才能自动地关闭NATS GRPC连接.
	go func() {
		err := node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()
	//重要！！！必须开启了KeepAlive才能在退出时让服务端那边自动地关闭NATS GRPC连接.
	go func() {
		err := node.KeepAlive(discovery.Node{
			DC:      conf.Global.Dc,
			Service: "test",
			NID:     node.NID,
			RPC: discovery.RPC{
				Protocol: discovery.NGRPC,
				Addr:     conf.Nats.URL,
				//Params:   map[string]string{"username": "foo", "password": "bar"},
			},
		})
		if err != nil {
			log.Errorf("isglb.Node.KeepAlive(%v) error %v", node.NID, err)
		}
	}()
	pub := sfu.NewPublisher(&node)
	// Create a video track
	videoopt := []string{
		"-f", "dshow",
		"-i", "video=" + device,
		"-vcodec", "libvpx",
		"-b:v", "3M",
		"-f", "ivf",
		"pipe:1",
	}
	ffmpegCmd := exec.Command(ffmpeg, videoopt...) //nolint
	_, ffmpegOut, err := util.GetStdPipes(ffmpegCmd)
	if err != nil {
		panic(err)
	}
	videoTrack, err := util.MakeIVFTrackFromStdout(ffmpegOut, webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8})
	if err != nil {
		panic(err)
	}
	pub.NeedTrack = func(AddTrack func(track webrtc.TrackLocal) (*webrtc.RTPSender, error)) error {
		log.Warnf("needTrack started")

		rtpSender, videoTrackErr := AddTrack(videoTrack)
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
	pub.Switch(nid, map[string]interface{}{}, &pb2.ClientNeededSession{Session: sid, User: uid})

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
