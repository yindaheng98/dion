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
		// Just do it, peer can dealing with Stable state
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

type Publisher struct {
	peer    *ion_sfu.PeerLocal
	pc      *webrtc.PeerConnection
	errCh   chan error
	ctx     context.Context
	cancel  context.CancelFunc
	onClose func(err error)
}

func NewPublisher(peer *ion_sfu.PeerLocal, pc *webrtc.PeerConnection) Publisher {
	ctx, cancel := context.WithCancel(context.Background())
	return Publisher{peer: peer, pc: pc, errCh: make(chan error, 16), ctx: ctx, cancel: cancel}
}

// Publish publish PeerConnection to PeerLocal.Subscriber
func (p Publisher) Publish(sid string) error {
	errCh := p.errCh
	p.pc.OnNegotiationNeeded(func() {
		offer, err := p.pc.CreateOffer(nil)
		if err != nil {
			log.Errorf("Cannot CreateOffer in pc: %+v", err)
			select {
			case errCh <- err:
			default:
			}
			return
		}
		err = p.pc.SetLocalDescription(offer)
		if err != nil {
			log.Errorf("Cannot SetLocalDescription to pc: %+v", err)
			select {
			case errCh <- err:
			default:
			}
			return
		}
		answer, err := p.peer.Answer(offer)
		if err != nil {
			log.Errorf("Cannot create Answer in peer: %+v", err)
			select {
			case errCh <- err:
			default:
			}
			return
		}
		err = p.pc.SetRemoteDescription(*answer)
		if err != nil {
			log.Errorf("Cannot SetRemoteDescription to pc: %+v", err)
			select {
			case errCh <- err:
			default:
			}
			return
		}
	})

	err := p.peer.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:       false,
		NoSubscribe:     true,
		NoAutoSubscribe: false,
	})
	if err != nil {
		return err
	}

	candidateSetting(p.pc, p.peer, p.errCh, rtc.Target_PUBLISHER)

	go p.logger()
	return err
}

func (p Publisher) logger() {
	for {
		select {
		case err := <-p.errCh:
			p.close(err)
		case <-p.ctx.Done():
			p.close(nil)
		}
	}
}

func (p Publisher) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	addTrack, err := p.pc.AddTrack(track)
	if err != nil {
		return nil, err
	}
	return addTrack, nil
}

func (p Publisher) close(err0 error) {
	p.cancel()
	err := p.peer.Close()
	if err != nil {
		log.Errorf("Error when closing peer in publisher: %+v", err)
	}
	err = p.pc.Close()
	if err != nil {
		log.Errorf("Error when closing pc in publisher: %+v", err)
	}
	if p.onClose != nil {
		p.onClose(err0)
	}
}

func (p Publisher) Close() {
	p.close(nil)
}

func (p Publisher) OnConnectionStateChange(f func(webrtc.PeerConnectionState)) {
	p.pc.OnConnectionStateChange(f)
}

func (p Publisher) OnICEConnectionStateChange(f func(webrtc.ICEConnectionState)) {
	p.pc.OnICEConnectionStateChange(f)
}
