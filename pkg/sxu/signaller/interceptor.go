package signaller

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/ion"
)

// PubIRFBuilderFactory is a factory to create ion_sfu.InterceptorRegistryFactoryBuilder
type PubIRFBuilderFactory interface {
	// NewBuilder create ion_sfu.InterceptorRegistryFactoryBuilder
	// remote is the node to  be connected
	NewBuilder(remote *ion.Node) ion_sfu.InterceptorRegistryFactoryBuilder
}
