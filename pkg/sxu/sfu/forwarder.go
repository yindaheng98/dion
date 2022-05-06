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
	ctx    context.Context
	cancel context.CancelFunc
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

func newRTC(track *Track, sfu *ion_sfu.SFU) (*rtc.RTC, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(track.ctx)
	r := rtc.NewRTC(sfu)
	r.OnError = func(err error) {
		_ = r.Close()
		select {
		case <-ctx.Done():
		default:
			log.Errorf("Forwarding exited with an error: %+v", err)
			cancel()
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
		if err != nil {
			_ = r.Close()
			select {
			case <-ctx.Done():
				return
			case <-time.After(RetryInterval):
				log.Errorf("Error when starting forward a track, retry it: %+v", err)
				cancel()
				continue
			}
		}
		for {
			// TODO: 想办法检查SFU里的Track，只在Track有差别的时候才发送更新请求
			// TODO: 需要从SFU端查询Track信息，应该hack SFU，在Track不变的时候返回一种特殊信息
			// TODO: 似乎RID就是Layer？
			if track.Track != nil {
				log.Debugf("Syncing track: %+v", track.Track)
				err := r.Update(track.Track)
				if err != nil {
					log.Errorf("Error syncing track, retry it: %+v", err)
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(RetryInterval):
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
	}
}
