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
}

func RandBool() bool {
	return rand.Intn(2) == 0
}

func RandNode(nid string) *ion.Node {
	return &ion.Node{
		Dc:      util.RandomString(2),
		Nid:     nid,
		Service: util.RandomString(4),
		Rpc: &ion.RPC{
			Protocol: util.RandomString(4),
			Addr:     util.RandomString(8),
		},
	}
}

func (Random) UpdateSFUStatus(current []*pb.SFUStatus, reports []*pb.QualityReport) (expected []*pb.SFUStatus) {
	fmt.Printf("┎Received status: %+v\n", current)
	fmt.Printf("┖Received report: %+v\n", reports)
	for _, s := range current {
		if RandBool() || RandBool() {
			expected = append(expected, s)
		}
		if !RandBool() {
			continue
		}

		for _, t := range s.ForwardTracks {
			if !RandBool() {
				continue
			}
			if RandBool() {
				t.TrackId = util.RandomString(4)
			}
			if RandBool() {
				t.Src = RandNode(t.Src.Nid)
			}
		}
		if RandBool() {
			s.ForwardTracks = append(s.ForwardTracks, &pb.ForwardTrack{
				Src:     RandNode(util.RandomString(1)),
				TrackId: util.RandomString(4),
			})
		}

		for _, t := range s.ProceedTracks {
			if !RandBool() {
				continue
			}
			if RandBool() {
				t.SrcTrackId = util.RandomString(4)
			}
			if RandBool() {
				t.DstTrackId = util.RandomString(4)
			}
			if RandBool() {
				t.Procedure = util.RandomString(4)
			}
		}
		if RandBool() {
			s.ProceedTracks = append(s.ProceedTracks, &pb.ProceedTrack{
				SrcTrackId: util.RandomString(4),
				DstTrackId: util.RandomString(4),
				Procedure:  util.RandomString(2),
			})
		}
	}

	if !RandBool() {
		expected = append(expected, &pb.SFUStatus{
			SFU: RandNode(util.RandomString(1)),
		})
	}

	return
}

type RandReports struct {
	reports []*pb.QualityReport
}

func (r *RandReports) RandReports() (reports []*pb.QualityReport) {
	for _, report := range r.reports {
		if RandBool() || RandBool() {
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
