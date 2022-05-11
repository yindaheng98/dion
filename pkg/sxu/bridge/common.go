package bridge

import (
	"context"
	log "github.com/pion/ion-log"
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
				log.Errorf("Cannot add ICECandidate: %+v", err)
				select {
				case errCh <- err:
				default:
				}
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
			select {
			case errCh <- err:
			default:
			}
			return
		}
	})
}

type bridgePeer struct {
	peer    *ion_sfu.PeerLocal
	pc      *webrtc.PeerConnection
	errCh   chan error
	ctx     context.Context
	cancel  context.CancelFunc
	OnClose func(err error)
}

func newPeer(pr *ion_sfu.PeerLocal, pc *webrtc.PeerConnection) bridgePeer {
	ctx, cancel := context.WithCancel(context.Background())
	return bridgePeer{peer: pr, pc: pc, errCh: make(chan error, 16), ctx: ctx, cancel: cancel}
}

func (p bridgePeer) logger() {
	for {
		select {
		case err := <-p.errCh:
			p.close(err)
		case <-p.ctx.Done():
			p.close(nil)
		}
	}
}

func (p bridgePeer) close(err0 error) {
	p.cancel()
	err := p.peer.Close()
	if err != nil {
		log.Errorf("Error when closing bridgePeer in publisher: %+v", err)
	}
	err = p.pc.Close()
	if err != nil {
		log.Errorf("Error when closing pc in publisher: %+v", err)
	}
	if p.OnClose != nil {
		p.OnClose(err0)
	}
}

func (p bridgePeer) Close() {
	p.close(nil)
}

func (p bridgePeer) OnConnectionStateChange(f func(webrtc.PeerConnectionState)) {
	p.pc.OnConnectionStateChange(f)
}

func (p bridgePeer) OnICEConnectionStateChange(f func(webrtc.ICEConnectionState)) {
	p.pc.OnICEConnectionStateChange(f)
}
