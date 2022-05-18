package syncer

import (
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// qualityReporter describe an abstract SFU that can report the running quality
type qualityReporter struct {
	t TransmissionReporter
	c ComputationReporter

	tCh chan *pb.TransmissionReport
	cCh chan *pb.ComputationReport
}

func newQualityReporter(t TransmissionReporter, c ComputationReporter) *qualityReporter {
	r := &qualityReporter{
		t:   t,
		c:   c,
		tCh: make(chan *pb.TransmissionReport, 1),
		cCh: make(chan *pb.ComputationReport, 1),
	}
	t.Bind(r.tCh)
	c.Bind(r.cCh)
	return r
}

// FetchReport fetch a quality report
// Block until return a new quality report
func (q *qualityReporter) FetchReport() *pb.QualityReport {
	select {
	case tr := <-q.tCh:
		return &pb.QualityReport{
			Timestamp: timestamppb.Now(),
			Report:    &pb.QualityReport_Transmission{Transmission: tr},
		}
	case cr := <-q.cCh:
		return &pb.QualityReport{
			Timestamp: timestamppb.Now(),
			Report:    &pb.QualityReport_Computation{Computation: cr},
		}
	}
}
