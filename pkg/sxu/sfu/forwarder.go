package sfu

import (
	"context"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"time"
)

const RetryInterval time.Duration = time.Second * 1

type Track struct {
	util.ForwardTrackItem
	cancel   context.CancelFunc
	updateCh chan util.ForwardTrackItem
}

type ForwardController struct {
	ForwardTrackRoutineFactory
	tracks map[string]*Track
}

func NewForwardController(factory ForwardTrackRoutineFactory) *ForwardController {
	return &ForwardController{
		ForwardTrackRoutineFactory: factory,
		tracks:                     make(map[string]*Track),
	}
}

func (f *ForwardController) StartForwardTrack(trackInfo *pb.ForwardTrack) {
	item := util.ForwardTrackItem{Track: trackInfo}
	if old, ok := f.tracks[item.Key()]; ok {
		f.ReplaceForwardTrack(old.Track, trackInfo)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	updateCh := make(chan util.ForwardTrackItem, 1)
	updateCh <- item
	track := &Track{
		ForwardTrackItem: item,
		cancel:           cancel,
		updateCh:         updateCh,
	}
	f.tracks[track.Key()] = track
	go f.ForwardTrackRoutine(ctx, updateCh) // One thread pre track
}

func (f *ForwardController) StopForwardTrack(trackInfo *pb.ForwardTrack) {
	item := util.ForwardTrackItem{Track: trackInfo}
	if old, ok := f.tracks[item.Key()]; ok {
		old.cancel()                 // Stop routine
		delete(f.tracks, item.Key()) // Delete track
	}
}

func (f *ForwardController) ReplaceForwardTrack(oldTrackInfo *pb.ForwardTrack, newTrackInfo *pb.ForwardTrack) {
	oldItem := util.ForwardTrackItem{Track: oldTrackInfo}
	newItem := util.ForwardTrackItem{Track: newTrackInfo}
	if oldItem.Key() != newItem.Key() { // if not from the same node
		f.StopForwardTrack(oldTrackInfo)  // Just stop the old
		f.StartForwardTrack(newTrackInfo) // And start a new
	} else if oldTrack, ok := f.tracks[oldItem.Key()]; !ok { // if not exist
		f.StartForwardTrack(newTrackInfo) // Just start a new
	} else { // From the same node and exist in current tracks
		oldTrack.ForwardTrackItem = newItem // Change it
		select {
		case oldTrack.updateCh <- newItem: // Send to the update channel
		default:
			select {
			case <-oldTrack.updateCh:
			default:
			}
			oldTrack.updateCh <- newItem
		}
	}
}
