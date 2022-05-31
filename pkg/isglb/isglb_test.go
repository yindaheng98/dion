package isglb

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"testing"
	"time"

	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/algorithms/impl/random"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
)

const sleep = 100
const N = 100

var conf = Config{
	Global: config.Global{Dc: "dc1"},
	Log:    config.LogConf{Level: "DEBUG"},
	Nats:   config.NatsConf{URL: "nats://192.168.94.131:4222"},
}

func TestISGLB(t *testing.T) {
	isglb := NewWithID("isglb-test", func() algorithms.Algorithm { return &random.Random{} })
	err := isglb.Start(conf)
	if err != nil {
		t.Error(err)
	}
	select {}
}

func TestISGLBClient(t *testing.T) {
	node := ion.NewNode("sxu-" + util.RandomString(6))
	err := node.Start(conf.Nats.URL)
	if err != nil {
		t.Error(err)
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
			Service: config.ServiceSXU,
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
	cli := NewISGLBClient(&node, "*", map[string]interface{}{})

	cli.OnSFUStatusRecv = func(ss *pb.SFUStatus) {
		t.Logf("Received SFUStatus: %s\n", ss.String())
	}
	cli.Connect()
	// ↑↑↑↑↑ Connect ↑↑↑↑↑

	// ↓↓↓↓↓ Generate and send Random Data ↓↓↓↓↓
	s := &pb.SFUStatus{
		SFU: random.RandNode(node.NID),
	}
	del := make([]*pb.SFUStatus, N)
	rr := &random.RandReports{}
	for i := 0; i < N; i++ {
		if random.RandBool() {
			t.Log("Sending a SFUStatus......")
			cli.SendSFUStatus(s)
			del[i] = s
			time.Sleep(sleep * time.Millisecond)
		} else {
			del[i] = &pb.SFUStatus{
				SFU: random.RandNode(node.NID),
			}
		}
		if random.RandBool() {
			random.RandChange(s)
		} else if random.RandBool() {
			s = &pb.SFUStatus{
				SFU: random.RandNode("sxu-" + util.RandomString(6)),
			}
		}
		for _, r := range rr.RandReports() {
			t.Log("Sending a Report......")
			cli.SendQualityReport(r)
			time.Sleep(sleep * time.Millisecond)
		}
	}
	time.Sleep(1 * time.Second)
	cli.Close()
	time.Sleep(1 * time.Second)
}
