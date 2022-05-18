package bridge

import (
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

func (e EntranceFactory) NewDoor() (util.UnblockedDoor, error) {
	sub, err := e.SubscriberFactory.NewDoor()
	if err != nil {
		return nil, err
	}
	return Entrance{
		Subscriber: sub.(Subscriber),
		exit:       e.exit,
		road:       e.road,
	}, nil
}

// Entrance of a Bridge
type Entrance struct {
	Subscriber           // Subscriber is its entrance, Entrance is also a Subscriber
	exit       Publisher // Publisher is its exit
	road       Processor

	sender *webrtc.RTPSender
}

func (p BridgePeer) SetOnConnectionStateChange(OnBroken func(error), OnConnected func()) {
	p.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		if state >= webrtc.ICEConnectionStateDisconnected {
			log.Errorf("ICEConnectionStateDisconnected")
			OnBroken(fmt.Errorf("ICEConnectionStateDisconnected %v", state))
		} else if state == webrtc.ICEConnectionStateConnected {
			log.Infof("ICEConnectionStateDisconnected")
			OnConnected()
		}
	})
	p.pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state >= webrtc.PeerConnectionStateDisconnected {
			log.Errorf("PeerConnectionStateDisconnected")
			OnBroken(fmt.Errorf("PeerConnectionStateDisconnected %v", state))
		} else if state == webrtc.PeerConnectionStateConnected {
			log.Infof("ICEConnectionStateDisconnected")
			OnConnected()
		}
	})
}

func (e Entrance) Lock(init util.Param, OnBroken func(badGay error)) error {
	sid := init.(SID)

	e.Subscriber.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		videoTrack := e.road.AddTrack(remote, receiver, OnBroken)

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

	return e.Subscriber.Lock(sid, OnBroken)
}

func (e Entrance) Repair(param util.Param) error {
	return fmt.Errorf("Entrance cannot be repaired ")
}

func (e Entrance) Update(param util.Param) error {
	return fmt.Errorf("Entrance cannot be updated ")
}

func (e Entrance) Remove() {
	if e.sender != nil {
		err := e.exit.RemoveTrack(e.sender)
		if err != nil {
			log.Errorf("Cannot remove track: %+v", err)
		}
	}
	e.Subscriber.Remove()
}
