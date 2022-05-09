package syncer

import (
	"math/rand"
	"testing"
	"time"

	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/util"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/algorithms/impl/random"
	"github.com/yindaheng98/dion/config"
	"github.com/yindaheng98/dion/pkg/isglb"
	pb "github.com/yindaheng98/dion/proto"
)

type TestQualityReporter struct {
	random.RandReports
}

func (t TestQualityReporter) FetchReport() *pb.QualityReport {
	<-time.After(time.Duration(rand.Int31n(10)) * time.Second)
	for {
		reports := t.RandReports.RandReports()
		if len(reports) > 0 {
			return reports[rand.Intn(len(reports))]
		}
	}
}

type TestSessionTracker struct {
}

func (t TestSessionTracker) FetchSessionEvent() *SessionEvent {
	<-time.After(time.Duration(rand.Int31n(1000)) * time.Millisecond)
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

func TestISGLBSyncer(t *testing.T) {
	ISGLB := isglb.New(RandomAlg)
	err := ISGLB.Start(isglb.Config{
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
	syncer := NewSFUStatusSyncer(
		&node, ISGLB.NID, random.RandNode(node.NID),
		ToolBox{
			TrackRouter:     nil,
			QualityReporter: TestQualityReporter{random.RandReports{}},
			SessionTracker:  TestSessionTracker{},
		},
	)
	syncer.Start()
	<-time.After(30 * time.Second)
	syncer.Stop()
	<-time.After(1 * time.Second)
	ISGLB.Close()
}
