package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

type Subscriber struct {
	bridgePeer
}

func NewSubscriber(peer *ion_sfu.PeerLocal, pc *webrtc.PeerConnection) Subscriber {
	return Subscriber{bridgePeer: newPeer(peer, pc)}
}

func (s Subscriber) Subscribe(sid string) error {
	errCh := s.errCh
	s.peer.OnOffer = func(offer *webrtc.SessionDescription) {
		err := s.pc.SetRemoteDescription(*offer)
		if err != nil {
			log.Errorf("Cannot SetRemoteDescription to pc: %+v", err)
			select {
			case errCh <- err:
			default:
			}
			return
		}
		answer, err := s.pc.CreateAnswer(nil)
		if err != nil {
			log.Errorf("Cannot CreateAnswer in pc: %+v", err)
			select {
			case errCh <- err:
			default:
			}
			return
		}
		err = s.peer.SetRemoteDescription(answer)
		if err != nil {
			log.Errorf("Cannot SetRemoteDescription in bridgePeer: %+v", err)
			select {
			case errCh <- err:
			default:
			}
			return
		}
	}

	err := s.peer.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:       true,
		NoSubscribe:     false,
		NoAutoSubscribe: false,
	})
	if err != nil {
		return err
	}

	candidateSetting(s.pc, s.peer, s.errCh, rtc.Target_SUBSCRIBER)

	go s.logger()
	return err
}

func (s Subscriber) OnTrack(f func(*webrtc.TrackRemote, *webrtc.RTPReceiver)) {
	s.pc.OnTrack(f)
}
