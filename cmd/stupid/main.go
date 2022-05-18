package main

import (
	"bufio"
	"flag"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
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
	var confFile, ffmpeg, testvideo, filter string
	flag.StringVar(&confFile, "conf", "cmd/stupid/sfu.toml", "sfu config file")
	flag.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "path to ffmpeg executable")
	flag.StringVar(&testvideo, "testvideo", "size=1280x720:rate=30", "ffmpeg -i testsrc=???")
	flag.StringVar(&filter, "filter", "drawtext=text='%{localtime\\:%Y-%M-%d %H.%m.%S}':fontsize=60:x=(w-text_w)/2:y=(h-text_h)/2", "ffmpeg -vf ???")

	flag.Parse()

	if confFile == "" {
		flag.PrintDefaults()
		return
	}
	conf := readConf(confFile)

	ffmpegOut := makeVideo(ffmpeg, testvideo, filter)

	log.Init(conf.Log.Level)
	log.Infof("--- starting sfu node ---")

	iSFU := ion_sfu.NewSFU(conf.Config)
	pub := NewPublisherFactory(ffmpegOut, iSFU)
	dog := util.NewWatchDogWithUnblockedDoor(pub)
	dog.Watch(bridge.SID(MyName))

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
