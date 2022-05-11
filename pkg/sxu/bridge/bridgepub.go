package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

type Publisher struct {
	bridgePeer
}

func NewPublisher(peer *ion_sfu.PeerLocal, pc *webrtc.PeerConnection) Publisher {
	return Publisher{bridgePeer: newPeer(peer, pc)}
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
			log.Errorf("Cannot create Answer in bridgePeer: %+v", err)
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

func (p Publisher) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	addTrack, err := p.pc.AddTrack(track)
	if err != nil {
		return nil, err
	}
	return addTrack, nil
}
