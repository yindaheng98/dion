package main

import (
	"bufio"
	"flag"
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/stupid"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var (
	conf = sfu.Config{}
	file string
)

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

func main() {
	var ffmpeg, testvideo, filter string
	flag.StringVar(&file, "conf", "cmd/stupid/sfu.toml", "sfu config file")
	flag.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "path to ffmpeg executable")
	flag.StringVar(&testvideo, "testvideo", "size=1280x720:rate=30", "ffmpeg -i testsrc=???")
	flag.StringVar(&filter, "filter", "drawtext=text='dion stupid':fontsize=60:x=(w-text_w)/2:y=(h-text_h)/2", "ffmpeg -vf ???")

	flag.Parse()

	if file == "" {
		flag.PrintDefaults()
		return
	}

	err := conf.Load(file)
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		flag.PrintDefaults()
		return
	}

	fmt.Printf("config %s load ok!\n", file)

	log.Init(conf.Log.Level)

	log.Infof("--- making video ---")

	ffmpegOut := makeVideo(ffmpeg, testvideo, filter)

	log.Infof("--- starting sfu node ---")

	server := stupid.New(ffmpegOut)
	if err := server.Start(conf); err != nil {
		panic(err)
	}
	defer server.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
