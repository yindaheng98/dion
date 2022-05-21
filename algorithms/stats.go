package algorithms

import (
	"github.com/pion/interceptor"
	"github.com/pion/ion/proto/ion"
	pb "github.com/yindaheng98/dion/proto"
)

type AtomReport interface{}

type SessionReport struct {
	SID, UID string
	Report   AtomReport
}

// ReportGathererBuilder : How to analysis SessionReport and give TransmissionReport?
type ReportGathererBuilder interface {
	// NewGatherer : get SessionReport from i and put TransmissionReport to o
	NewGatherer(src, dst *ion.Node, i <-chan SessionReport, o chan<- *pb.TransmissionReport)
}

// ReporterInterceptorFactory just a interceptor.Factory for ReporterInterceptor
type ReporterInterceptorFactory interface {
	NewInterceptor(id string) (ReporterInterceptor, error)
}

// ReporterInterceptor : How to analysis the RTP/RTCP packet and give report?
type ReporterInterceptor interface {
	interceptor.Interceptor

	// BindReportChannel : put report to o
	BindReportChannel(o chan<- AtomReport)
}
