package main

import (
	"flag"
	"fmt"
	"github.com/yindaheng98/dion/pkg/sxu"

	"os"
	"os/signal"
	"syscall"

	log "github.com/pion/ion-log"
	"github.com/spf13/viper"
)

var (
	conf = sxu.Config{}
	file string
)

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func load() bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		return false
	}
	err = viper.UnmarshalExact(&conf)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", file, err)
		return false
	}
	fmt.Printf("config %s load ok!\n", file)
	return true
}

func parse() bool {
	flag.StringVar(&file, "c", "configs/sfu.toml", "config file")

	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !load() {
		return false
	}

	if *help {
		showHelp()
		return false
	}
	return true
}

func main() {
	if !parse() {
		showHelp()
		os.Exit(-1)
	}

	log.Init(conf.Log.Level)

	log.Infof("--- starting isglb node ---")
	node := sxu.New(nil) // TODO
	if err := node.Start(conf); err != nil {
		log.Errorf("isglb start error: %v", err)
		os.Exit(-1)
	}
	defer node.Close()

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
