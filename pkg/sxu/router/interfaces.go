package router

import (
	"context"
	"github.com/yindaheng98/dion/util"
)

type ForwardTrackRoutineFactory interface {
	// ForwardTrackRoutine forward a track according to the item in updateCh
	// Should retry until the ctx exit
	ForwardTrackRoutine(ctx context.Context, updateCh <-chan util.ForwardTrackItem, init util.ForwardTrackItem)
}
