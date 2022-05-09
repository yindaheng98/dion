package sfu

import (
	"context"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pbrtc "github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/pkg/sxu/rtc"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/grpc/metadata"
	"time"
)

const RetryInterval time.Duration = time.Second * 1

type Track struct {
	util.ForwardTrackItem
	ctx      context.Context
	cancel   context.CancelFunc
	updateCh chan util.ForwardTrackItem
}

type ForwardController struct {
	sfu      *ion_sfu.SFU
	client   pbrtc.RTCClient
	tracks   map[string]*Track
	Metadata metadata.MD
}

func NewForwarder(sfu *ion_sfu.SFU, client pbrtc.RTCClient) *ForwardController {
	return &ForwardController{
		sfu:    sfu,
		client: client,
		tracks: make(map[string]*Track),
	}
}

func (f *ForwardController) StartForwardTrack(trackInfo *pb.ForwardTrack) {
	item := util.ForwardTrackItem{Track: trackInfo}
	if old, ok := f.tracks[item.Key()]; ok {
		f.ReplaceForwardTrack(old.Track, trackInfo)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	track := &Track{
		ForwardTrackItem: item,
		ctx:              ctx,
		cancel:           cancel,
		updateCh:         make(chan util.ForwardTrackItem, 1),
	}
	f.tracks[track.Key()] = track
	go f.forwardTrackRoutine(track) // One thread pre track
}

func newRTC(track *Track, sfu *ion_sfu.SFU) (*rtc.RTC, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(track.ctx)
	r := rtc.NewRTC(sfu)
	r.OnError = func(err error) {
		_ = r.Close() // Close
		cancel()      // Close
		select {
		case <-track.ctx.Done(): // this track should exit?
			return // just exit
		default: // should not exit?
			log.Errorf("Forwarding exited with an error: %+v", err) // should retry
		}
	}
	return r, ctx, cancel
}

// forwardTrackRoutine retry until success
// !!!SINGLE THREAD for each Track!!!
func (f *ForwardController) forwardTrackRoutine(track *Track) {
	for {
		r, ctx, cancel := newRTC(track, f.sfu)
		err := r.Start(track.Track.RemoteSessionId, track.Track.LocalSessionId, f.client, f.Metadata)
		if err != nil { // if error
			_ = r.Close() // Close
			cancel()      // Close
			select {
			case <-track.ctx.Done(): // this track should exit?
				return // exit
			case <-time.After(RetryInterval): // this track should not exit
				log.Errorf("Error when starting forward a track, retry it: %+v", err)
				continue // retry
			}
		}
		// Start successfully, the start updating
		retryItemCh := make(chan util.ForwardTrackItem, 1)
		for {
			var item util.ForwardTrackItem
			select {
			case <-ctx.Done(): // some error occurred? updating should not continue
				return
			case item = <-track.updateCh: // get item from update channel or retry channel
			case item = <-retryItemCh: // get item from update channel or retry channel
			}
			if r.IsSame(item.Track) { // If is same
				continue // Just skip
			}
			log.Debugf("Updating track: %+v", item.Track)
			err := r.Update(item.Track) // Update it
			if err != nil {
				select {
				case <-ctx.Done(): // Error occurred? updating should not continue
					return
				default: // should retry
					log.Errorf("Error updating track, retry it: %+v", err)
					retryItemCh <- item
					select {
					case <-ctx.Done(): // Error occurred? updating should not continue
						return
					case <-time.After(RetryInterval): // Delay to retry
					}
				}
			}
		}
	}
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
