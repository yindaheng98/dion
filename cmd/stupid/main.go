package main

import (
	"flag"
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/stupid"
	"os"
	"os/signal"
	"syscall"
)

var (
	conf = sfu.Config{}
	file string
)

func main() {
	var ffmpeg, testvideo, filter string
	flag.StringVar(&file, "conf", "cmd/sxu/sfu.toml", "sfu config file")
	flag.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "path to ffmpeg executable")
	flag.StringVar(&testvideo, "testvideo", "size=1280x720:rate=30", "ffmpeg -i testsrc=???")
	flag.StringVar(&filter, "filter", "drawtext=text='%{localtime\\:%Y-%m-%d %H.%M.%S}':fontsize=60:x=(w-text_w)/2:y=(h-text_h)/2", "ffmpeg -vf ???")

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

	log.Infof("--- starting sfu node ---")

	server := stupid.New(ffmpeg)
	server.Testsrc, server.Filter = testvideo, filter
	if err := server.Start(conf); err != nil {
		panic(err)
	}
	defer server.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
