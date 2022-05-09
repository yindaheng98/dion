package router

import (
	"context"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
)

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

func replace(oldItemInList *Track, newItem util.ForwardTrackItem) {
	// And replace the new
	oldItemInList.ForwardTrackItem = newItem // Change it
	select {
	case oldItemInList.updateCh <- newItem: // Send to the update channel
	default:
		select {
		case <-oldItemInList.updateCh:
		default:
		}
		select {
		case oldItemInList.updateCh <- newItem:
		default:
		}
	}
}

func (f *ForwardController) ReplaceForwardTrack(oldTrackInfo *pb.ForwardTrack, newTrackInfo *pb.ForwardTrack) {
	oldItem := util.ForwardTrackItem{Track: oldTrackInfo}
	newItem := util.ForwardTrackItem{Track: newTrackInfo}
	newItemInList, newExist := f.tracks[newItem.Key()]
	if oldItem.Key() != newItem.Key() { // if not from the same node
		_, oldExist := f.tracks[oldItem.Key()]
		if oldExist {
			if newExist {
				// old item and new item both exists?
				f.StopForwardTrack(oldTrackInfo) // Should stop the old
				replace(newItemInList, newItem)  // And replace the newItemInList
			} else {
				// old item exist but new item not exists?
				f.StopForwardTrack(oldTrackInfo)  // Just stop the old
				f.StartForwardTrack(newTrackInfo) // And start a new
			}
		} else {
			if newExist {
				// old item not exists and new item exist?
				replace(newItemInList, newItem) // And replace the newItemInList
			} else {
				// old item and new item both not exists?
				f.StartForwardTrack(newTrackInfo) // Just start a new
			}
		}
	} else { // if from the same node
		if newExist { // exists?
			replace(newItemInList, newItem) // Just replace the newItemInList
		} else {
			f.StartForwardTrack(newTrackInfo) // Just start a new
		}
	}
}
