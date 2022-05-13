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
}

func (e Entrance) Lock(init util.Param, OnBroken func(badGay error)) error {
	sid := init.(SID)

	e.entr.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

		// Set the handler for ICE connection state
		// This will notify you when the peer has connected/disconnected
		e.entr.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			log.Debugf("Connection State has changed %s \n", connectionState.String())
			if connectionState == webrtc.ICEConnectionStateConnected {
				iceConnectedCtxCancel()
			}
		})

		// Set the handler for Peer connection state
		// This will notify you when the peer has connected/disconnected
		e.entr.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
			log.Debugf("Peer Connection State has changed: %s\n", s.String())

			if s == webrtc.PeerConnectionStateFailed {
				// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
				// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
				// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
				log.Errorf("Peer Connection has gone to failed exiting")
				OnBroken(fmt.Errorf("PeerConnectionStateFailed"))
			}
		})

		videoTrack := e.road.AddTrack(iceConnectedCtx, remote, receiver)

		rtpSender, videoTrackErr := e.exit.AddTrack(videoTrack)
		if videoTrackErr != nil {
			OnBroken(videoTrackErr)
			return
		}

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
	e.entr.Remove()
}
