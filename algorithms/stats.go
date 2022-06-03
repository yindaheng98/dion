package algorithms

import (
	"github.com/pion/interceptor"
	"github.com/pion/ion/proto/ion"
	pb "github.com/yindaheng98/dion/proto"
)

type SessionReport[AtomReport any] struct {
	SID, UID string
	Report   AtomReport
}

// ReportGathererBuilder : How to analysis SessionReport and give TransmissionReport?
type ReportGathererBuilder[AtomReport any] interface {
	// NewGatherer : get SessionReport from i and put TransmissionReport to o
	NewGatherer(src, dst *ion.Node, i <-chan SessionReport[AtomReport], o chan<- *pb.TransmissionReport)
}

// ReporterInterceptorFactory just a interceptor.Factory for ReporterInterceptor
type ReporterInterceptorFactory[AtomReport any] interface {
	NewInterceptor(id string) (ReporterInterceptor[AtomReport], error)
}

// ReporterInterceptor : How to analysis the RTP/RTCP packet and give report?
type ReporterInterceptor[AtomReport any] interface {
	interceptor.Interceptor

	// BindReportChannel : put report to o
	BindReportChannel(o chan<- AtomReport)
}
