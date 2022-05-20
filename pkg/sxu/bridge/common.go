package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
)

type BridgePeer struct {
	peer *ion_sfu.PeerLocal
	pc   *webrtc.PeerConnection
}

func NewBridgePeer(peer *ion_sfu.PeerLocal, pc *webrtc.PeerConnection) BridgePeer {
	return BridgePeer{peer: peer, pc: pc}
}

func (p BridgePeer) setOnICECandidateForPeer(OnBroken func(error), Target rtc.Target) (addCandidate func()) {
	var pcCand []webrtc.ICECandidateInit // Store unsended ICECandidate
	addCandidate = func() {
		tpcCand := pcCand
		pcCand = []webrtc.ICECandidateInit{} // Clear it
		for _, c := range tpcCand {
			err := p.pc.AddICECandidate(c)
			if err != nil {
				log.Errorf("Cannot add ICECandidate: %+v", err)
				OnBroken(err)
				return
			}
		}
	}
	p.peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
		if target != int(Target) { // detect target
			return // I do not want other's candidate
		}
		if p.pc.CurrentRemoteDescription() == nil { // If not initialized
			pcCand = append(pcCand, *candidate) // just store it
			return
		}
		// initialized
		tpcCand := pcCand
		pcCand = []webrtc.ICECandidateInit{} // Clear it
		// And add them all
		tpcCand = append(tpcCand, *candidate)
		for _, c := range tpcCand {
			err := p.pc.AddICECandidate(c)
			if err != nil {
				log.Errorf("Cannot add ICECandidate: %+v", err)
				OnBroken(err)
				return
			}
		}
	}
	return addCandidate
}

func (p BridgePeer) setOnICECandidateForPC(OnBroken func(error), Target rtc.Target) {
	p.pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		// Just do it, BridgePeer can dealing with Stable state
		if candidate == nil {
			return
		}
		err := p.peer.Trickle(candidate.ToJSON(), int(Target))
		if err != nil {
			log.Errorf("Cannot Trickle: %+v", err)
			OnBroken(err)
			return
		}
	})
}

func (p BridgePeer) setOnIceCandidate(OnBroken func(error), Target rtc.Target) (addCandidate func()) {
	addCandidate = p.setOnICECandidateForPeer(OnBroken, Target)
	p.setOnICECandidateForPC(OnBroken, Target)
	return
}
func (p BridgePeer) Remove() {
	err := p.peer.Close()
	if err != nil {
		log.Errorf("Error when closing peer in BridgePeer: %+v", err)
	}
	err = p.pc.Close()
	if err != nil {
		log.Errorf("Error when closing pc in BridgePeer: %+v", err)
	}
}

type SID string

func (s SID) Clone() util.Param {
	return s
}
