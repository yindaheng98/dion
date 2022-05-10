package bridge

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

func candidateSetting(pc *webrtc.PeerConnection, peer *ion_sfu.PeerLocal, errCh chan<- error, Target rtc.Target) {
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
				errCh <- err
				return
			}
		}
	}
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		// Just do it, peer can dealing with Stable state
		err := peer.Trickle(candidate.ToJSON(), int(Target))
		if err != nil {
			errCh <- err
			return
		}
	})
}

type Subscriber struct {
	peer ion_sfu.PeerLocal
	pc   webrtc.PeerConnection
}

// Subscribe subscribe PeerConnection from PeerLocal
func (p *Subscriber) Subscribe(sid string) error {
	// subscribe from PeerLocal, so i should interact with Subscriber
	errCh := make(chan error)
	p.peer.OnOffer = func(sdp *webrtc.SessionDescription) {
		err := p.pc.SetRemoteDescription(*sdp)
		if err != nil {
			errCh <- err
			return
		}
		answer, err := p.pc.CreateAnswer(nil)
		if err != nil {
			errCh <- err
			return
		}
		err = p.peer.SetRemoteDescription(answer)
		if err != nil {
			errCh <- err
			return
		}
	}

	candidateSetting(&p.pc, &p.peer, errCh, rtc.Target_SUBSCRIBER)

	err := p.peer.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:       true,
		NoSubscribe:     false,
		NoAutoSubscribe: false,
	})
	if err != nil {
		return err
	}
	err, _ = <-errCh
	return err
}

type Publisher struct {
	peer ion_sfu.PeerLocal
	pc   webrtc.PeerConnection
}

// Publish publish PeerConnection to PeerLocal.Subscriber
func (p *Publisher) Publish(sid string) error {
	errCh := make(chan error)
	p.pc.OnNegotiationNeeded(func() {
		offer, err := p.pc.CreateOffer(nil)
		if err != nil {
			errCh <- err
			return
		}
		err = p.pc.SetLocalDescription(offer)
		if err != nil {
			errCh <- err
			return
		}
		answer, err := p.peer.Answer(offer)
		if err != nil {
			errCh <- err
			return
		}
		err = p.pc.SetRemoteDescription(*answer)
		if err != nil {
			errCh <- err
			return
		}
	})

	candidateSetting(&p.pc, &p.peer, errCh, rtc.Target_PUBLISHER)

	err := p.peer.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:       false,
		NoSubscribe:     true,
		NoAutoSubscribe: false,
	})
	if err != nil {
		return err
	}
	err, _ = <-errCh
	return err
}
