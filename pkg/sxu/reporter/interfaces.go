package reporter

import (
	"github.com/pion/ion/proto/ion"
	pb "github.com/yindaheng98/dion/proto"
)

// ReportGatherer is to gather report
type ReportGathererBuilder interface {
	// BindIO : this chan will give you the report
	NewGatherer(src, dst *ion.Node, i <-chan SessionReport, o chan<- *pb.TransmissionReport)
}

type SessionReport struct {
	sid, uid string
	report   AtomReport
}
