package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
)

func candidateSetting(pc *webrtc.PeerConnection, peer *ion_sfu.PeerLocal, OnBroken func(error), Target rtc.Target) {
	var pcCand []webrtc.ICECandidateInit // Store unsended ICECandidate
	peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
		if target != int(Target) { // detect target
			return // I do not want other's candidate
		}
		if pc.CurrentRemoteDescription() == nil { // If not initialized
			pcCand = append(pcCand, *candidate) // just store it
			return
		}
		// initialized
		tpcCand := pcCand
		pcCand = []webrtc.ICECandidateInit{} // Clear it
		// And add them all
		tpcCand = append(tpcCand, *candidate)
		for _, c := range tpcCand {
			err := pc.AddICECandidate(c)
			if err != nil {
				log.Errorf("Cannot add ICECandidate: %+v", err)
				OnBroken(err)
				return
			}
		}
	}
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		// Just do it, bridgePeer can dealing with Stable state
		if candidate == nil {
			return
		}
		err := peer.Trickle(candidate.ToJSON(), int(Target))
		if err != nil {
			log.Errorf("Cannot Trickle: %+v", err)
			OnBroken(err)
			return
		}
	})
}

type bridgePeer struct {
	peer *ion_sfu.PeerLocal
	pc   *webrtc.PeerConnection
}

func (p bridgePeer) Remove() {
	err := p.peer.Close()
	if err != nil {
		log.Errorf("Error when closing bridgePeer in publisher: %+v", err)
	}
	err = p.pc.Close()
	if err != nil {
		log.Errorf("Error when closing pc in publisher: %+v", err)
	}
}

func (p bridgePeer) OnConnectionStateChange(f func(webrtc.PeerConnectionState)) {
	p.pc.OnConnectionStateChange(f)
}

func (p bridgePeer) OnICEConnectionStateChange(f func(webrtc.ICEConnectionState)) {
	p.pc.OnICEConnectionStateChange(f)
}

type SID string

func (s SID) Clone() util.Param {
	return s
}
