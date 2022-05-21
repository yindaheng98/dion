package sfu

import (
	"github.com/pion/interceptor"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
)

// PubIRFBuilder is a factory to create ion_sfu.InterceptorRegistryFactoryBuilder
type PubIRFBuilder struct{}

func (PubIRFBuilder) Build(sid string, uid string) ion_sfu.InterceptorRegistryFactory {
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
		return interceptorRegistry
	}
}
