package sfu

import (
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"sync"
)

type candidates struct {
	sync.Mutex
	candidates []webrtc.ICECandidateInit
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
