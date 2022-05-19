package signaller

import (
	"github.com/pion/interceptor"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/ion"
)

// PubInterceptorFactory is a factory to create *interceptor.Registry
type PubInterceptorFactory interface {
	// NewRegistry create *interceptor.Registry
	// remote is the node to  be connected
	NewRegistry(remote *ion.Node) *interceptor.Registry
}

type StupidPubInterceptorFactory struct {
}

func (s StupidPubInterceptorFactory) NewRegistry(remote *ion.Node) *interceptor.Registry {
	log.Warnf("No PubInterceptorFactory specified for %+v", remote)
	return nil
}
