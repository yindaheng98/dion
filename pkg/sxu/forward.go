package sxu

import (
	"context"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pbrtc "github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/pkg/sxu/rtc"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/grpc/metadata"
	"time"
)

const RetryInterval time.Duration = time.Second * 1

type ForwardTrackRoutineFactory struct {
	sfu      *ion_sfu.SFU
	client   pbrtc.RTCClient
	Metadata metadata.MD
}

func newRTC(Ctx context.Context, sfu *ion_sfu.SFU) (*rtc.RTC, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(Ctx)
	// 大Ctx退出表示ForwardTrackRoutine应该退出
	// 小ctx表示ForwardTrackRoutine出错，应该重试
	r := rtc.NewRTC(sfu)
	r.OnError = func(err error) {
		_ = r.Close() // Close
		cancel()      // Close
		select {
		case <-Ctx.Done(): // this track should exit?
			return // just exit
		default: // should not exit?
			log.Errorf("Forwarding exited with an error: %+v", err) // should retry
		}
	}
	return r, ctx, cancel
}

func (f ForwardTrackRoutineFactory) ForwardTrackRoutine(Ctx context.Context, updateCh <-chan util.ForwardTrackItem) {
	retryItemCh := make(chan util.ForwardTrackItem, 1)
	for {
		var item util.ForwardTrackItem
		select {
		case <-Ctx.Done(): // this forwarding should exit
			return
		case item = <-updateCh: // get item from update channel or retry channel
		default: // 这是在实现带优先级的select
			select {
			case <-Ctx.Done(): // this forwarding should exit
				return
			case item = <-updateCh: // get item from update channel or retry channel
			case item = <-retryItemCh: // get item from update channel or retry channel
			}
		}

		log.Infof("Starting track forward: %+v", item.Track)
		// init the forwarding
		r, ctx, cancel := newRTC(Ctx, f.sfu)
		err := r.Start(item.Track.RemoteSessionId, item.Track.LocalSessionId, f.client, f.Metadata)
		if err != nil { // if error
			_ = r.Close() // Close
			cancel()      // Close
			select {
			case <-Ctx.Done(): // this track should exit?
				return // exit
			case <-time.After(RetryInterval): // this track should not exit
				log.Errorf("Error when starting forward a track, retry it: %+v", err)
				retryItemCh <- item // retry it
				continue            // retry it
			}
		}

		// syncing(updating) the forwarding
	L:
		for {
			select {
			case <-ctx.Done(): // this updating should not continue
				break L
			case item = <-updateCh: // get item from update channel or retry channel
			default: // 这是在实现带优先级的select
				select {
				case <-ctx.Done(): // this forwarding should exit
					break L
				case item = <-updateCh: // get item from update channel or retry channel
				case item = <-retryItemCh: // get item from update channel or retry channel
				}
			}
			if r.IsSame(item.Track.Tracks) { // If is same
				continue // Just skip
			}
			log.Debugf("Updating track forward: %+v", item.Track)
			err := r.Update(item.Track.Tracks) // Update it
			if err != nil {
				select {
				case <-ctx.Done(): // Error occurred? updating should not continue
					break L
				case <-time.After(RetryInterval): // Delay to retry
					log.Errorf("Error updating track, retry it: %+v", err)
					retryItemCh <- item
				}
			}
		}
	}
}
