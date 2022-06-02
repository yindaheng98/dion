package sfu

import (
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"sync"
)

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
		if candidate == nil {
			log.Warnf("OnICECandidate give a nil")
			return
		}
		err := sub.SendTrickle(candidate, pb.Target_SUBSCRIBER)
		if err != nil {
			log.Errorf("Cannot SendTrickle: %+v", err)
		}
	})

	pc.OnTrack(sub.OnTrack)
	return pc, nil
}

type candidates struct {
	sync.Mutex
	candidates []webrtc.ICECandidateInit
}

// negotiate sub negotiate
func (sub *Subscriber) negotiate(pc *webrtc.PeerConnection, c *candidates, sdp webrtc.SessionDescription) error {
	log.Debugf("[S=>C] negotiate sdp=%v", sdp)
	// 1.sub set remote sdp
	err := pc.SetRemoteDescription(sdp)
	if err != nil {
		log.Errorf("negotiate pc.SetRemoteDescription err=%v", err)
		return err
	}

	// 3. safe to add candidate after SetRemoteDescription
	c.Lock()
	if len(c.candidates) > 0 {
		for _, candidate := range c.candidates {
			log.Debugf("pc.AddICECandidate candidate=%v", candidate)
			_ = pc.AddICECandidate(candidate)
		}
		c.candidates = []webrtc.ICECandidateInit{}
	}
	c.Unlock()

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
	return sub.SendAnswer(answer)
}

func (sub *Subscriber) trickle(pc *webrtc.PeerConnection, c *candidates, candidate webrtc.ICECandidateInit, target pb.Target) {
	if target != pb.Target_SUBSCRIBER {
		log.Warnf("[S=>C] candidate=%v target=%v", candidate, target)
		return
	}
	log.Debugf("[S=>C] candidate=%v target=%v", candidate, target)

	if pc.CurrentRemoteDescription() == nil {
		c.candidates = append(c.candidates, candidate)
	} else {
		err := pc.AddICECandidate(candidate)
		if err != nil {
			log.Errorf("Cannot AddICECandidate err=%v", err)
		}
	}

}
