package random

import (
	"fmt"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/isglb/algorithms"
	pb "github.com/yindaheng98/isglb/proto"
	"math/rand"
)

// Random is a node selection algorithm, just for test
type Random struct {
	algorithms.Algorithm
	nodes map[string]*pb.SFUStatus
}

func RandBool() bool {
	return rand.Intn(2) == 0
}

func RandNode(nid string) *ion.Node {
	return &ion.Node{
		Nid:     nid,
		Service: util.RandomString(4),
	}
}

func RandChange(s *pb.SFUStatus) *pb.SFUStatus {
	s.SFU.Service = util.RandomString(4)
	return s
}

func (r *Random) UpdateSFUStatus(current []*pb.SFUStatus, reports []*pb.QualityReport) (expected []*pb.SFUStatus) {
	fmt.Printf("┎Received status: %+v\n", current)
	fmt.Printf("┖Received report: %+v\n", reports)
	if r.nodes == nil {
		r.nodes = map[string]*pb.SFUStatus{}
	}
	for _, s := range current {
		r.nodes[s.SFU.Nid] = s
		if RandBool() {
			s = RandChange(s)
		}
		if RandBool() {
			expected = append(expected, s)
		}
	}

	if RandBool() {
		expected = append(expected, &pb.SFUStatus{
			SFU: RandNode("test-" + util.RandomString(6)),
		})
	}

	for _, s := range r.nodes {
		if RandBool() {
			s = RandChange(s)
		}
		if RandBool() {
			expected = append(expected, s)
		}
	}
	fmt.Printf("▶  Return status: %+v\n", expected)
	return
}

type RandReports struct {
	reports []*pb.QualityReport
}

func (r *RandReports) RandReports() (reports []*pb.QualityReport) {
	for _, report := range r.reports {
		if RandBool() || RandBool() || RandBool() {
			reports = append(reports, report)
		}
		if RandBool() {
			continue
		}
		if RandBool() {
			t := &timestamp.Timestamp{}
			t.Seconds = rand.Int63()
			report.Timestamp = t
		}
	}
	if RandBool() {
		t := &timestamp.Timestamp{}
		t.Seconds = rand.Int63()
		reports = append(reports, &pb.QualityReport{
			Timestamp: t,
		})
	}
	r.reports = reports
	return
}
