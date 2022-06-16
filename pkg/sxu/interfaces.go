package sxu

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/islb"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
)

type ToolBoxBuilder interface {
	Build(sxu *SXU, node *islb.Node, sfu *ion_sfu.SFU) syncer.ToolBox
}

type WithOption func(*syncer.ToolBox, *islb.Node, *ion_sfu.SFU)

type DefaultToolBoxBuilder struct {
	with []WithOption
}

func NewDefaultToolBoxBuilder(with ...WithOption) DefaultToolBoxBuilder {
	return DefaultToolBoxBuilder{with: with}
}

func (b DefaultToolBoxBuilder) Build(sxu *SXU, node *islb.Node, sfu *ion_sfu.SFU) syncer.ToolBox {
	t := syncer.ToolBox{}
	for _, w := range b.with {
		w(&t, node, sfu)
	}
	if t.TrackForwarder == nil {
		WithTrackForwarder()(&t, node, sfu)
	}
	if t.TrackProcessor == nil {
		t.TrackProcessor = syncer.StupidTrackProcesser{}
	}
	if t.SessionTracker == nil {
		WithSessionTracker(sxu)(&t, node, sfu)
	}
	if t.TransmissionReporter == nil {
		t.TransmissionReporter = &syncer.StupidTransmissionReporter{}
	}
	if t.ComputationReporter == nil {
		t.ComputationReporter = &syncer.StupidComputationReporter{}
	}
	return t
}

func WithSessionTracker(sxu *SXU) WithOption {
	return func(box *syncer.ToolBox, node *islb.Node, sfu *ion_sfu.SFU) {
		sxu.s.TrackSession = true
		box.SessionTracker = sxu.s
	}
}
