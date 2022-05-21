package signaller

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/ion"
)

// PubInterceptorFactory is a factory to create *interceptor.Registry
type PubInterceptorFactory interface {
	// NewFactory create ion_sfu.PeerInterceptorFactory
	// remote is the node to  be connected
	NewFactory(remote *ion.Node) ion_sfu.PeerInterceptorFactory
}

type StupidPubInterceptorFactory struct {
}

func (s StupidPubInterceptorFactory) NewRegistry(remote *ion.Node) ion_sfu.PeerInterceptorFactory {
	log.Warnf("No PubInterceptorFactory specified for %+v", remote)
	return nil
}
