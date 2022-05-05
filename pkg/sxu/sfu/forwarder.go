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
	"sync"
	"time"
)

const RetryInterval time.Duration = time.Second * 1

type Track struct {
	util.ForwardTrackItem
	ctx    context.Context
	cancel context.CancelFunc

	r   *rtc.RTC
	rMu sync.Mutex // TODO: Single threaded the assessment
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
	}
	f.tracks[track.Key()] = track
	go f.forwardTrackRoutine(track)
}

// forwardTrackRoutine retry until success
func (f *ForwardController) forwardTrackRoutine(track *Track) {
	for {
		track.rMu.Lock()
		track.r = nil
		r := rtc.NewRTC(f.sfu)
		r.OnError = func(err error) {
			track.rMu.Lock()
			track.r = nil
			track.rMu.Unlock()
			_ = r.Close()
			select {
			case <-track.ctx.Done():
			case <-time.After(RetryInterval):
				log.Errorf("Forwarding exited with an error, retry it: %+v", err)
				go f.forwardTrackRoutine(track)
			}
		}
		err := r.Start(track.Track.RemoteSessionId, track.Track.LocalSessionId, f.client, f.Metadata)
		if err != nil {
			_ = r.Close()
			select {
			case <-time.After(RetryInterval):
				log.Errorf("Error when forwarding a track, retry it: %+v", err)
				continue
			case <-track.ctx.Done():
			}
		}
		track.r = r
		track.rMu.Unlock()
		break
	}
}

func (f *ForwardController) StopForwardTrack(trackInfo *pb.ForwardTrack) {
	item := util.ForwardTrackItem{Track: trackInfo}
	if old, ok := f.tracks[item.Key()]; ok {
		old.cancel()                 // Stop routine
		delete(f.tracks, item.Key()) // Delete track
	}
}

func (f *ForwardController) syncTrackRoutine(track *Track) {

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
	}
}
