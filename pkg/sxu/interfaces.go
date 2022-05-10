package sxu

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/yindaheng98/dion/pkg/sxu/router"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
)

type ToolBoxBuilder interface {
	Build(node *ion.Node, sfu *ion_sfu.SFU) syncer.ToolBox
}

type DefaultToolBoxBuilder struct {
}

func (b DefaultToolBoxBuilder) Build(node *ion.Node, sfu *ion_sfu.SFU) syncer.ToolBox {
	return syncer.ToolBox{
		TrackForwarder: router.NewForwardRouter(NewForwardTrackRoutineFactory(node, sfu)),
	}
}
