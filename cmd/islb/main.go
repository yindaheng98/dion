package main

import (
	"flag"
	"fmt"
	"github.com/yindaheng98/dion/config"

	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/pkg/islb"
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

	log.Infof("--- starting islb node ---")
	node := islb.NewISLB()
	if err := node.Start(conf); err != nil {
		log.Errorf("islb start error: %v", err)
		os.Exit(-1)
	}
	defer node.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
