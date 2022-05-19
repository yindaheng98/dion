package signaller

func WithPubInterceptorFactory(pubIrFact PubInterceptorFactory) func(SignallerFactory) {
	return func(factory SignallerFactory) {
		factory.irFact = pubIrFact
	}
}
