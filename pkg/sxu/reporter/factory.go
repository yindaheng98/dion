package reporter

import (
	"github.com/pion/interceptor"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/ion"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/pkg/sxu/signaller"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	pb "github.com/yindaheng98/dion/proto"
)

// InterceptorReporter 通过interceptor收集数据
type InterceptorReporter struct {
	syncer.TransmissionReporter    // 这主要是一个TransmissionReporter用于收集数据
	signaller.PubIRFBuilderFactory // 这主要是一个PubIRFBuilderFactory用于创建interceptor
	// 于是从interceptor里收集数据

	local *ion.Node
	gb    algorithms.ReportGathererBuilder
	ch    chan<- *pb.TransmissionReport

	r algorithms.ReporterInterceptorFactory
}

func NewInterceptorReporter(local *ion.Node, gb algorithms.ReportGathererBuilder, r algorithms.ReporterInterceptorFactory) InterceptorReporter {
	return InterceptorReporter{
		local: local,
		gb:    gb,
		r:     r,
	}
}

// Bind 是给TransmissionReporter绑定输出channel
func (t InterceptorReporter) Bind(reports chan<- *pb.TransmissionReport) {
	t.ch = reports
}

// NewBuilder 是给PubIRFBuilderFactory创建interceptor
func (t InterceptorReporter) NewBuilder(remote *ion.Node) ion_sfu.InterceptorRegistryFactoryBuilder {
	ch := make(chan algorithms.SessionReport, 16)
	t.gb.NewGatherer(remote, t.local, ch, t.ch)                           // 启动收集器
	return transmissionReporterIRFBuilder{remote: remote, ch: ch, r: t.r} // 创建下级收集器
}

// transmissionReporterIRFBuilder 就是TransmissionReporterIRFBuilderFactory输出的ion_sfu.InterceptorRegistryFactoryBuilder
// 对每个uid输出一个自定义的ion_sfu.InterceptorRegistryFactory
type transmissionReporterIRFBuilder struct {
	ion_sfu.InterceptorRegistryFactoryBuilder // 这是一个ion_sfu.InterceptorRegistryFactoryBuilder

	remote *ion.Node
	r      algorithms.ReporterInterceptorFactory
	ch     chan<- algorithms.SessionReport
}

func (t transmissionReporterIRFBuilder) Build(sid, uid string) ion_sfu.InterceptorRegistryFactory {
	return func(mediaEngine *webrtc.MediaEngine, config ion_sfu.WebRTCTransportConfig) *interceptor.Registry {
		interceptorRegistry := &interceptor.Registry{}
		if err := webrtc.ConfigureNack(mediaEngine, interceptorRegistry); err != nil {
			log.Errorf("Cannot ConfigureNack: %+v", err)
		}
		if err := webrtc.ConfigureRTCPReports(interceptorRegistry); err != nil {
			log.Errorf("Cannot ConfigureNack: %+v", err)
		}
		if err := webrtc.ConfigureTWCCSender(mediaEngine, interceptorRegistry); err != nil {
			log.Errorf("Cannot ConfigureNack: %+v", err)
		}
		ch := make(chan algorithms.AtomReport, 16)
		// 和ReportGathererBuilder.NewGatherer差不多的功能
		go func(sid, uid string, o chan<- algorithms.SessionReport, i <-chan algorithms.AtomReport) {
			for {
				ar, ok := <-i
				if !ok {
					return
				}
				o <- algorithms.SessionReport{
					SID:    sid,
					UID:    uid,
					Report: ar,
				}
			}
		}(sid, uid, t.ch, ch)
		interceptorRegistry.Add(interceptorFactory{
			r:  t.r,
			ch: ch,
		})
		return interceptorRegistry
	}
}

// 每个remote node 分配一个reporterInterceptorFactory
type interceptorFactory struct {
	r  algorithms.ReporterInterceptorFactory
	ch chan<- algorithms.AtomReport
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
