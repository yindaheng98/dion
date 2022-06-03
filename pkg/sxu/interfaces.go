package sxu

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/islb"
	"github.com/yindaheng98/dion/pkg/sxu/room"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	pb2 "github.com/yindaheng98/dion/proto"
)

type ToolBoxBuilder interface {
	Build(node *islb.Node, sfu *ion_sfu.SFU) syncer.ToolBox
}

type WithOption func(*syncer.ToolBox, *islb.Node, *ion_sfu.SFU)

type DefaultToolBoxBuilder struct {
	with []WithOption
}

func NewDefaultToolBoxBuilder(with ...WithOption) DefaultToolBoxBuilder {
	return DefaultToolBoxBuilder{with: with}
}

func (b DefaultToolBoxBuilder) Build(node *islb.Node, sfu *ion_sfu.SFU) syncer.ToolBox {
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
		WithSessionTracker()(&t, node, sfu)
	}
	if t.TransmissionReporter == nil {
		t.TransmissionReporter = &syncer.StupidTransmissionReporter{}
	}
	if t.ComputationReporter == nil {
		t.ComputationReporter = &syncer.StupidComputationReporter{}
	}
	return t
}

func WithSessionTracker() WithOption {
	return func(box *syncer.ToolBox, node *islb.Node, sfu *ion_sfu.SFU) {
		s := room.NewService()
		box.SessionTracker = s
		pb2.RegisterRoomServer(node.ServiceRegistrar(), s)
	}
}
