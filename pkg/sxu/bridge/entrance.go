package bridge

import (
	"context"
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
)

type EntranceFactory struct {
	SubscriberFactory
	exit Publisher
	road Processor
}

func (e EntranceFactory) NewDoor() (util.Door, error) {
	entr, err := e.SubscriberFactory.NewDoor()
	if err != nil {
		return nil, err
	}
	return Entrance{
		entr: entr.(Subscriber),
		exit: e.exit,
		road: e.road,
	}, nil
}

type Entrance struct {
	entr Subscriber
	exit Publisher
	road Processor

	sender *webrtc.RTPSender
}

func (e Entrance) Lock(init util.Param, OnBroken func(badGay error)) error {
	sid := init.(SID)

	e.entr.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())
		e.entr.SetOnConnectionStateChange(OnBroken, iceConnectedCtxCancel)

		videoTrack := e.road.AddTrack(iceConnectedCtx, remote, receiver)

		rtpSender, videoTrackErr := e.exit.AddTrack(videoTrack)
		if videoTrackErr != nil {
			OnBroken(videoTrackErr)
			return
		}

		e.sender = rtpSender

		// Read incoming RTCP packets
		// Before these packets are returned they are processed by interceptors. For things
		// like NACK this needs to be called.
		go func() {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
					return
				}
			}
		}()
	})

	return e.entr.Lock(sid, OnBroken)
}

func (e Entrance) Repair(param util.Param, OnBroken func(badGay error)) error {
	return fmt.Errorf("Entrance cannot be repaired ")
}

func (e Entrance) Update(param util.Param, OnBroken func(badGay error)) error {
	return fmt.Errorf("Entrance cannot be updated ")
}

func (e Entrance) Remove() {
	if e.sender != nil {
		err := e.exit.RemoveTrack(e.sender)
		if err != nil {
			log.Errorf("Cannot remove track: %+v", err)
		}
	}
	e.entr.Remove()
}
