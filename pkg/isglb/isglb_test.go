package isglb

import (
	"fmt"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/isglb/algorithms"
	"github.com/yindaheng98/isglb/config"
	pb "github.com/yindaheng98/isglb/proto"
	"testing"
	"time"
)
import "github.com/yindaheng98/isglb/algorithms/impl/random"

const sleep = 1000

func TestISGLB(t *testing.T) {
	isglb := New(func() algorithms.Algorithm { return &random.Random{} })
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
	cli := NewISGLBClient(&node, node.NID, map[string]interface{}{})

	cli.OnSFUStatusRecv = func(ss *pb.SFUStatus) {
		t.Log(fmt.Printf("Received SFU status: %s", ss.String()))
	}
	cli.Connect()
	// ↑↑↑↑↑ Connect ↑↑↑↑↑

	// ↓↓↓↓↓ Generate and send Random Data ↓↓↓↓↓
	s := &pb.SFUStatus{
		SFU: random.RandNode(node.NID),
	}
	rr := &random.RandReports{}
	for i := 0; i < 100; i++ {
		if random.RandBool() {
			err := cli.SendSyncRequest(&pb.SyncRequest{Request: &pb.SyncRequest_Status{Status: s}})
			if err != nil {
				t.Error(err)
			}
			time.Sleep(sleep * time.Millisecond)
		}
		if random.RandBool() {
			random.RandChange(s)
		} else if random.RandBool() {
			s = &pb.SFUStatus{
				SFU: random.RandNode("sxu-" + util.RandomString(6)),
			}
		}
		for _, r := range rr.RandReports() {
			err := cli.SendSyncRequest(&pb.SyncRequest{Request: &pb.SyncRequest_Report{Report: r}})
			if err != nil {
				t.Error(err)
			}
			time.Sleep(sleep * time.Millisecond)
		}
	}
	time.Sleep(1 * time.Second)
	cli.Close()
	time.Sleep(1 * time.Second)
}
