package sfu

import (
	"context"
	"github.com/yindaheng98/dion/util"
)

type ForwardTrackRoutineFactory interface {
	ForwardTrackRoutine(ctx context.Context, updateCh <-chan util.ForwardTrackItem)
}
