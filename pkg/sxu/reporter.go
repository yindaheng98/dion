package sxu

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
type InterceptorReporter[AtomReport any] struct {
	syncer.TransmissionReporter    // 这主要是一个TransmissionReporter用于收集数据
	signaller.PubIRFBuilderFactory // 这主要是一个PubIRFBuilderFactory用于创建interceptor
	// 于是从interceptor里收集数据

	local *ion.Node
	gb    algorithms.ReportGathererBuilder[AtomReport]
	ch    chan<- *pb.TransmissionReport

	r algorithms.ReporterInterceptorFactory[AtomReport]
}

func NewInterceptorReporter[AtomReport any](
	local *ion.Node,
	gb algorithms.ReportGathererBuilder[AtomReport],
	r algorithms.ReporterInterceptorFactory[AtomReport],
) InterceptorReporter[AtomReport] {
	return InterceptorReporter[AtomReport]{
		local: local,
		gb:    gb,
		r:     r,
	}
}

// Bind 是给TransmissionReporter绑定输出channel
func (t InterceptorReporter[AtomReport]) Bind(reports chan<- *pb.TransmissionReport) {
	t.ch = reports
}

// NewBuilder 是给PubIRFBuilderFactory创建interceptor
func (t InterceptorReporter[AtomReport]) NewBuilder(remote *ion.Node) ion_sfu.InterceptorRegistryFactoryBuilder {
	ch := make(chan algorithms.SessionReport[AtomReport], 16)
	t.gb.NewGatherer(remote, t.local, ch, t.ch)                                       // 启动收集器
	return transmissionReporterIRFBuilder[AtomReport]{remote: remote, ch: ch, r: t.r} // 创建下级收集器
}

// transmissionReporterIRFBuilder 就是TransmissionReporterIRFBuilderFactory输出的ion_sfu.InterceptorRegistryFactoryBuilder
// 对每个uid输出一个自定义的ion_sfu.InterceptorRegistryFactory
type transmissionReporterIRFBuilder[AtomReport any] struct {
	ion_sfu.InterceptorRegistryFactoryBuilder // 这是一个ion_sfu.InterceptorRegistryFactoryBuilder

	remote *ion.Node
	r      algorithms.ReporterInterceptorFactory[AtomReport]
	ch     chan<- algorithms.SessionReport[AtomReport]
}

func (t transmissionReporterIRFBuilder[AtomReport]) Build(sid, uid string) ion_sfu.InterceptorRegistryFactory {
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
		ch := make(chan AtomReport, 16)
		// 和ReportGathererBuilder.NewGatherer差不多的功能
		go func(sid, uid string, o chan<- algorithms.SessionReport[AtomReport], i <-chan AtomReport) {
			for {
				ar, ok := <-i
				if !ok {
					return
				}
				o <- algorithms.SessionReport[AtomReport]{
					SID:    sid,
					UID:    uid,
					Report: ar,
				}
			}
		}(sid, uid, t.ch, ch)
		interceptorRegistry.Add(interceptorFactory[AtomReport]{
			r:  t.r,
			ch: ch,
		})
		return interceptorRegistry
	}
}

// 每个remote node 分配一个reporterInterceptorFactory
type interceptorFactory[AtomReport any] struct {
	r  algorithms.ReporterInterceptorFactory[AtomReport]
	ch chan<- AtomReport
}

// NewInterceptor 每次连接都要生成一个Interceptor
func (r interceptorFactory[AtomReport]) NewInterceptor(id string) (interceptor.Interceptor, error) {
	ri, err := r.r.NewInterceptor(id)
	if err != nil {
		return nil, err
	}
	ri.BindReportChannel(r.ch)
	return ri, nil
}
