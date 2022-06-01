package sxu

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/pkg/sxu/router"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	"github.com/yindaheng98/dion/util/ion"
)

type ToolBoxBuilder interface {
	Build(node *ion.Node, sfu *ion_sfu.SFU) syncer.ToolBox
}

type WithOption func(*syncer.ToolBox, *ion.Node, *ion_sfu.SFU)

type DefaultToolBoxBuilder struct {
	with []WithOption
}

func NewDefaultToolBoxBuilder(with ...WithOption) DefaultToolBoxBuilder {
	return DefaultToolBoxBuilder{with: with}
}

func (b DefaultToolBoxBuilder) Build(node *ion.Node, sfu *ion_sfu.SFU) syncer.ToolBox {
	t := syncer.ToolBox{}
	for _, w := range b.with {
		w(&t, node, sfu)
	}
	if t.TrackForwarder == nil {
		t.TrackForwarder = syncer.StupidTrackForwarder{}
	}
	if t.TrackProcessor == nil {
		t.TrackProcessor = syncer.StupidTrackProcesser{}
	}
	if t.SessionTracker == nil {
		t.SessionTracker = syncer.StupidSessionTracker{}
	}
	if t.TransmissionReporter == nil {
		t.TransmissionReporter = &syncer.StupidTransmissionReporter{}
	}
	if t.ComputationReporter == nil {
		t.ComputationReporter = &syncer.StupidComputationReporter{}
	}
	return t
}

func WithProcessorFactory(pro algorithms.ProcessorFactory) WithOption {
	return func(box *syncer.ToolBox, node *ion.Node, sfu *ion_sfu.SFU) {
		if pro != nil {
			box.TrackProcessor = router.NewProceedRouter(sfu, pro)
		}
	}
}

func WithTrackForwarder(with ...func(router.ForwardRouter)) WithOption {
	return func(box *syncer.ToolBox, node *ion.Node, sfu *ion_sfu.SFU) {
		TrackForwarder := router.NewForwardRouter(sfu, NewNRPCConnPool(node))
		for _, w := range with {
			w(TrackForwarder)
		}
		box.TrackForwarder = TrackForwarder
	}
}

func WithTransmissionReporter(reporter syncer.TransmissionReporter) WithOption {
	return func(box *syncer.ToolBox, node *ion.Node, sfu *ion_sfu.SFU) {
		if reporter != nil {
			box.TransmissionReporter = reporter
		}
	}
}

func WithComputationReporter(reporter syncer.ComputationReporter) WithOption {
	return func(box *syncer.ToolBox, node *ion.Node, sfu *ion_sfu.SFU) {
		if reporter != nil {
			box.ComputationReporter = reporter
		}
	}
}

func WithSessionTracker(tracker syncer.SessionTracker) WithOption {
	return func(box *syncer.ToolBox, node *ion.Node, sfu *ion_sfu.SFU) {
		if tracker != nil {
			box.SessionTracker = tracker
		}
	}
}
