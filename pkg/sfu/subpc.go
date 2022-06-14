package sfu

import (
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
)

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
