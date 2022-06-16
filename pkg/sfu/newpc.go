package sfu

import (
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
)

func newPeerConnection(SendTrickle func(candidate *webrtc.ICECandidate) error) (*webrtc.PeerConnection, error) {
	me := webrtc.MediaEngine{}
	err := me.RegisterDefaultCodecs()
	if err != nil {
		log.Errorf("Cannot RegisterDefaultCodecs %v", err)
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.stunprotocol.org:3478", "stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		log.Errorf("Cannot NewPeerConnection %v", err)
		return nil, err
	}

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Infof("ICEConnectionState %v", state.String())
	})

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			log.Warnf("OnICECandidate give a nil")
			return
		}
		err := SendTrickle(candidate)
		if err != nil {
			log.Errorf("Cannot SendTrickle: %+v", err)
		}
	})

	return pc, nil
}

func (sub *Subscriber) newPeerConnection() (*webrtc.PeerConnection, error) {
	pc, err := newPeerConnection(sub.SendTrickle)
	if err != nil {
		return nil, err
	}
	pc.OnTrack(sub.OnTrack)
	return pc, nil
}

func (pub *Publisher) newPeerConnection() (*webrtc.PeerConnection, error) {
	return newPeerConnection(pub.SendTrickle)
}
