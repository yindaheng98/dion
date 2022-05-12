package main

import (
	"bufio"
	"flag"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	"github.com/yindaheng98/dion/util"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

const MyName = "stupid"

// makeVideo Make a video
func makeVideo(ffmpegPath, param string) io.ReadCloser {
	videoopt := []string{
		"-f", "lavfi",
		"-i", "testsrc=" + param,
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

func main() {
	var confFile, session, ffmpeg, testvideo string
	flag.StringVar(&confFile, "conf", "cmd/stupid/sfu.toml", "sfu config file")
	flag.StringVar(&session, "session", MyName, "session of the video")
	flag.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "path to ffmpeg executable")
	flag.StringVar(&testvideo, "testvideo", "size=1280x720:rate=30", "size of the video")

	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}
	conf := readConf(confFile)

	ffmpegOut := makeVideo(ffmpeg, testvideo)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)
	pub := NewPublisherFactory(ffmpegOut, iSFU)
	dog := util.NewWatchDog(pub)
	dog.Watch(bridge.SID(MyName))

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
