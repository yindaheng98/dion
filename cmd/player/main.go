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
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"os"
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
	var ffplay, nid, sid, uid string
	flag.StringVar(&ffplay, "ffplay", "ffplay", "path to ffmpeg executable")
	flag.StringVar(&nid, "nid", "stupid", "target node id")
	flag.StringVar(&sid, "sid", "stupid", "target session id")
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
	sub := sfu.NewSubscriber(&node)
	sub.OnTrack = func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Warnf("onTrack started: %+v", remote)

		for {
			// Read RTP packets being sent to Pion
			_, _, readErr := remote.ReadRTP()
			fmt.Println("TestSubscriberFactory get a RTP Packet")
			if readErr != nil {
				fmt.Printf("TestSubscriberFactory RTP Packet read error %+v\n", readErr)
				return
			}
		}
	}
	sub.Switch(nid, map[string]interface{}{}, &pb.ClientNeededSession{
		Session: sid,
		User:    uid,
	})

	// Press Ctrl+C to exit the process
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
