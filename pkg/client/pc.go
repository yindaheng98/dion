package client

import (
	"errors"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

func (sub *Subscriber) setReconnect() {
	sub.client.OnReconnect = func() {
		pc, err := sub.newPeerConnection()
		if err != nil {
			return
		}
		sub.pc.Store(pc)
		sub.SendJoin(sub.session.Session, sub.session.User, map[string]string{})
	}
}

// negotiate sub negotiate
func (sub *Subscriber) negotiate(sdp webrtc.SessionDescription) error {
	pct := sub.pc.Load()
	if pct == nil {
		log.Warnf("No pc, cannot negotiate")
		return errors.New("No pc, cannot negotiate ")
	}
	pc := pct.(*webrtc.PeerConnection)
	log.Debugf("[S=>C] negotiate sdp=%v", sdp)
	// 1.sub set remote sdp
	err := pc.SetRemoteDescription(sdp)
	if err != nil {
		log.Errorf("negotiate pc.SetRemoteDescription err=%v", err)
		return err
	}

	// 3. safe to add candidate after SetRemoteDescription
	sub.recvCandMu.Lock()
	if len(sub.recvCandidates) > 0 {
		for _, candidate := range sub.recvCandidates {
			log.Debugf("pc.AddICECandidate candidate=%v", candidate)
			_ = pc.AddICECandidate(candidate)
		}
		sub.recvCandidates = []webrtc.ICECandidateInit{}
	}
	sub.recvCandMu.Unlock()

	// 4. create answer after add ice candidate
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Errorf("pc.CreateAnswer err=%v", err)
		return err
	}

	// 5. set local sdp(answer)
	err = pc.SetLocalDescription(answer)
	if err != nil {
		log.Errorf("pc.SetLocalDescription err=%v", err)
		return err
	}

	// 6. send answer to sfu
	sub.SendAnswer(answer)
	return nil
}

func (sub *Subscriber) newPeerConnection() (*webrtc.PeerConnection, error) {
	me := webrtc.MediaEngine{}
	err := me.RegisterDefaultCodecs()
	if err != nil {
		log.Errorf("Cannot RegisterDefaultCodecs %v", err)
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Errorf("Cannot NewPeerConnection %v", err)
		return nil, err
	}

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Infof("ICEConnectionState %v", state.String())
	})

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		sub.SendTrickle(candidate, pb.Target_SUBSCRIBER)
	})

	pc.OnTrack(sub.OnTrack)
	return pc, nil
}
