package sxu

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/pkg/islb"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
)

type ProcessorReporter interface {
	algorithms.ProcessorFactory
	syncer.ComputationReporter
	// TODO: 实现一个 algorithms.ProcessorFactory + syncer.ComputationReporter 即可实现计算层面的汇报
}

func WithProcessorReporter(pro ProcessorReporter) WithOption {
	return func(box *syncer.ToolBox, node *islb.Node, sfu *ion_sfu.SFU) {
		if pro != nil {
			box.TrackProcessor = NewProceedRouter(sfu, pro)
			box.ComputationReporter = pro
		}
	}
}
