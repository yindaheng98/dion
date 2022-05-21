package reporter

import (
	"github.com/pion/interceptor"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
)

type AtomReport interface{}

type ReporterInterceptorFactory interface {
	NewInterceptor(id string) (ReporterInterceptor, error)
}

// ReporterInterceptor is an interceptor that can report its status
type ReporterInterceptor interface {
	interceptor.Interceptor

	// BindReportChannel : when have something to report, put it into this chan
	BindReportChannel(chan<- AtomReport)
}

type reporterInterceptorFactory struct {
	r ReporterInterceptorFactory
}

func (r reporterInterceptorFactory) NewInterceptor(id string) (interceptor.Interceptor, error) {
	ri, err := r.r.NewInterceptor(id)
	if err != nil {
		return nil, err
	}
	return ri, nil
}

// ReportGatherer is to gather report
type ReportGatherer interface {
	syncer.TransmissionReporter

	// BindInputChannel : this chan will give you the report
	BindInputChannel(<-chan AtomReport)
}
