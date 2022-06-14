package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/islb"
	"github.com/yindaheng98/dion/pkg/sfu"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"io"
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
		log.Infof("OnTrack started: %+v\n", remote)
		ffplay := exec.Command(ffplay, "-f", "ivf", "-i", "pipe:0")
		stdin, stdout, err := util.GetStdPipes(ffplay)
		if err != nil {
			panic(err)
		}
		defer ffplay.Process.Kill()
		go func(stdout io.ReadCloser) {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}(stdout)
		ivfWriter, err := ivfwriter.NewWith(stdin)
		if err != nil {
			panic(err)
		}

		for {
			// Read RTP packets being sent to Pion
			rtp, _, readErr := remote.ReadRTP()
			log.Infof("Subscriber get a RTP Packet")
			if readErr != nil {
				log.Errorf("Subscriber RTP Packet read error %+v", readErr)
				return
			}

			if ivfWriterErr := ivfWriter.WriteRTP(rtp); ivfWriterErr != nil {
				log.Errorf("RTP Packet write error: %+v", ivfWriterErr)
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
