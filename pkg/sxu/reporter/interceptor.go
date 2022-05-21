package reporter

import "github.com/pion/interceptor"

type AtomReport interface{}

// 这里是构造interceptor的代码

// ReporterInterceptorFactory 将被封装为一个interceptor.Factory
type ReporterInterceptorFactory interface {
	NewInterceptor(id string) (ReporterInterceptor, error)
}

// ReporterInterceptor is an interceptor that can report its status
type ReporterInterceptor interface {
	interceptor.Interceptor

	// BindReportChannel : when have something to report, put it into this chan
	BindReportChannel(chan<- AtomReport)
}

// 每个remote node 分配一个reporterInterceptorFactory
type interceptorFactory struct {
	r  ReporterInterceptorFactory
	ch chan<- AtomReport
}

// NewInterceptor 每次连接都要生成一个Interceptor
func (r interceptorFactory) NewInterceptor(id string) (interceptor.Interceptor, error) {
	ri, err := r.r.NewInterceptor(id)
	if err != nil {
		return nil, err
	}
	ri.BindReportChannel(r.ch)
	return ri, nil
}
