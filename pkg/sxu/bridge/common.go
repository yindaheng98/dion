package bridge

import (
	"fmt"
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

func (p BridgePeer) SetOnConnectionStateChange(OnBroken func(error)) {
	p.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		if state >= webrtc.ICEConnectionStateDisconnected {
			log.Errorf("ICEConnectionStateDisconnected")
			OnBroken(fmt.Errorf("ICEConnectionStateDisconnected %v", state))
		}
	})
	p.pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state >= webrtc.PeerConnectionStateDisconnected {
			log.Errorf("PeerConnectionStateDisconnected")
			OnBroken(fmt.Errorf("PeerConnectionStateDisconnected %v", state))
		}
	})
}

func (p BridgePeer) SetOnIceCandidate(OnBroken func(error), Target rtc.Target) (addCandidate func()) {
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
	return
}

func (p BridgePeer) Remove() {
	err := p.peer.Close()
	if err != nil {
		log.Errorf("Error when closing BridgePeer in publisher: %+v", err)
	}
	err = p.pc.Close()
	if err != nil {
		log.Errorf("Error when closing pc in publisher: %+v", err)
	}
}

type SID string

func (s SID) Clone() util.Param {
	return s
}
