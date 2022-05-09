package random

import (
	"fmt"
	"math/rand"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/dion/algorithms"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
)

// Random is a node selection algorithm, just for test
type Random struct {
	algorithms.Algorithm
	nodes       map[string]*pb.SFUStatus
	RandomTrack bool
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
		if !r.RandomTrack && RandBool() { // change SFUStatus.SFU only when not random change track
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
		if !r.RandomTrack && RandBool() {
			s = RandChange(s)
		}
		if RandBool() {
			expected = append(expected, s)
		}
	}

	if r.RandomTrack {
		for _, s := range expected {
			if RandBool() {
				s.ForwardTracks = RandChangeForwardTracks(s.ForwardTracks)
			}
			if RandBool() {
				s.ProceedTracks = RandChangeProceedTracks(s.ProceedTracks)
			}
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
