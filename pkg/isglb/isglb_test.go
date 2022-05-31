package isglb

import (
	"fmt"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"math/rand"
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

func TestISGLB(t *testing.T) {
	isglb := NewWithID("isglb-test", func() algorithms.Algorithm { return &random.Random{} })
	err := isglb.Start(Config{
		Global: config.Global{Dc: "dc1"},
		Log:    config.LogConf{Level: "DEBUG"},
		Nats:   config.NatsConf{URL: "nats://192.168.94.131:4222"},
	})
	if err != nil {
		t.Error(err)
	}

	node := ion.NewNode("sxu-" + util.RandomString(6))
	err = node.Start("nats://192.168.94.131:4222")
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
	cli := NewISGLBClient(&node, node.NID, map[string]interface{}{})

	cli.OnSFUStatusRecv = func(ss *pb.SFUStatus) {
		t.Log(fmt.Printf("Received SFU status: %s\n", ss.String()))
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
			cli.SendQualityReport(r)
			time.Sleep(sleep * time.Millisecond)
		}

		if random.RandBool() {
			s := del[rand.Intn(i+1)]
			rpc := discovery.RPC{}
			if s.SFU.Rpc != nil {
				rpc = discovery.RPC{
					Protocol: discovery.Protocol(s.SFU.Rpc.Protocol),
					Addr:     s.SFU.Rpc.Addr,
				}
			}
			d := discovery.Node{
				DC:      s.SFU.Dc,
				Service: s.SFU.Service,
				NID:     s.SFU.Nid,
				RPC:     rpc,
			}
			isglb.s.handleNodeAction(discovery.Delete, d)
		}

		if i == N/4 {
			isglb.Close()
			t.Log("Stop it!!!!!!!!!!!!!!!!")
		}
		if i == N/2 {
			isglb = NewWithID("isglb-test", func() algorithms.Algorithm { return &random.Random{} })
			err := isglb.Start(Config{
				Global: config.Global{Dc: "dc1"},
				Log:    config.LogConf{Level: "DEBUG"},
				Nats:   config.NatsConf{URL: "nats://192.168.94.131:4222"},
			})
			if err != nil {
				t.Error(err)
			}
			t.Log("Restart it!!!!!!!!!!!!!!!!")
		}
	}
	time.Sleep(1 * time.Second)
	cli.Close()
	time.Sleep(1 * time.Second)
}
