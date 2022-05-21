package router

import "github.com/yindaheng98/dion/pkg/sxu/signaller"

func WithPubIRFBuilderFactory(irfbf signaller.PubIRFBuilderFactory) func(ForwardRouter) {
	return func(r ForwardRouter) {
		r.factory.IRFBF = irfbf
	}
}
