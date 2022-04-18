package isglb

import (
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/util"
	ion_pb "github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/isglb/algorithms"
	pb "github.com/yindaheng98/isglb/proto"
	"testing"
)
import "github.com/yindaheng98/isglb/algorithms/impl/random"

func TestISGLB(t *testing.T) {
	isglb := New(func() algorithms.Algorithm { return &random.Random{} })
	err := isglb.Start(Config{
		Global: global{Dc: "dc1"},
		Log:    logConf{Level: "DEBUG"},
		Nats:   natsConf{URL: "nats://192.168.1.2:4222"},
	})
	if err != nil {
		t.Error(err)
	}

	node := ion.NewNode("sxu-" + util.RandomString(6))
	err = node.Start("nats://192.168.1.2:4222")
	if err != nil {
		t.Error(err)
	}
	cli := NewISGLBClient(&node, isglb.NID, map[string]interface{}{})

	s := &pb.SFUStatus{
		SFU: &ion_pb.Node{
			Dc:      "dc1",
			Nid:     node.NID,
			Service: "sxu",
			Rpc: &ion_pb.RPC{
				Protocol: util.RandomString(4),
				Addr:     util.RandomString(4),
			},
		},
		ForwardTracks: []*pb.ForwardTrack{},
		ProceedTracks: []*pb.ProceedTrack{},
	}
	cli.OnSFUStatusRecv = func(ss *pb.SFUStatus) {
		if random.RandBool() {
			random.RandChange(s)
		}
		t.Log(ss.String())
	}
	cli.Connect()

	rr := &random.RandReports{}
	for i := 0; i < 100; i++ {
		if random.RandBool() {
			err := cli.SendSyncRequest(&pb.SyncRequest{Request: &pb.SyncRequest_Status{Status: s}})
			if err != nil {
				t.Error(err)
			}
		}
		for _, r := range rr.RandReports() {
			err := cli.SendSyncRequest(&pb.SyncRequest{Request: &pb.SyncRequest_Report{Report: r}})
			if err != nil {
				t.Error(err)
			}
		}
	}
}
