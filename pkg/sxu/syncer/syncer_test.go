package syncer

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"math/rand"
	"testing"
	"time"

	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/algorithms/impl/random"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/isglb"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util/ion"
)

const sleep = 1000

type TestTransmissionReporter struct {
	random.RandTransmissionReport
}

func (t TestTransmissionReporter) Bind(ch chan<- *pb.TransmissionReport) {
	go func(ch chan<- *pb.TransmissionReport) {
		for {
			<-time.After(time.Duration(rand.Int31n(sleep)) * time.Millisecond)
			ch <- t.RandReport()
		}
	}(ch)
}

type TestComputationReporter struct {
	random.RandComputationReport
}

func (t TestComputationReporter) Bind(ch chan<- *pb.ComputationReport) {
	go func(ch chan<- *pb.ComputationReport) {
		for {
			<-time.After(time.Duration(rand.Int31n(sleep)) * time.Millisecond)
			ch <- t.RandReport()
		}
	}(ch)
}

type TestSessionTracker struct {
}

func (t TestSessionTracker) FetchSessionEvent() *SessionEvent {
	<-time.After(time.Duration(rand.Int31n(sleep)) * time.Millisecond)
	return &SessionEvent{
		Session: &pb.ClientNeededSession{
			Session: "",
			User:    "",
		}, State: SessionEvent_State(rand.Intn(2)),
	}
}

func RandomAlg() algorithms.Algorithm {
	alg := &random.Random{}
	alg.RandomTrack = true
	return alg
}

var conf = isglb.Config{
	Global: config.Global{Dc: "dc1"},
	Log:    config.LogConf{Level: "DEBUG"},
	Nats:   config.NatsConf{URL: "nats://192.168.94.131:4222"},
}

func TestISGLB(t *testing.T) {
	ISGLB := isglb.New(RandomAlg)
	err := ISGLB.Start(conf)
	if err != nil {
		t.Error(err)
	}
	select {}
}

func TestISGLBSyncer(t *testing.T) {
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
	syncer := NewSFUStatusSyncer(
		&node, "*", random.RandNode(node.NID),
		ToolBox{
			TransmissionReporter: TestTransmissionReporter{random.RandTransmissionReport{}},
			ComputationReporter:  TestComputationReporter{random.RandComputationReport{}},
			SessionTracker:       TestSessionTracker{},
		},
	)
	syncer.Start()
	<-time.After(300 * time.Second)
	syncer.Stop()
	<-time.After(1 * time.Second)
}
