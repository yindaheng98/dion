package subscriber

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
)

type Subscriber struct {
	peer *UpPeerLocal
	sig  rtc.RTC_SignalClient
}

func NewSubscriber(peer *ion_sfu.PeerLocal, sig rtc.RTC_SignalClient) *Subscriber {
	return &Subscriber{
		peer: &UpPeerLocal{PeerLocal: peer},
		sig:  sig,
	}
}

// JoinLocal join a local session
func (s *Subscriber) JoinLocal(sid string) error {
	return s.peer.Join(sid)
}

type JoinConfig map[string]string

// JoinRemote join a remote session
func (s *Subscriber) JoinRemote(sid string, cfg ...*JoinConfig) error {
	offer, err := s.peer.CreateOffer()
	if err != nil {
		return err
	}

	var config map[string]string
	if len(cfg) > 0 {
		config = *cfg[0]
	} else {
		config = nil
	}
	return s.sig.Send(
		&rtc.Request{
			Payload: &rtc.Request_Join{
				Join: &rtc.JoinRequest{
					Sid:    sid,
					Uid:    s.peer.ID(),
					Config: config,
					Description: &rtc.SessionDescription{
						Target: rtc.Target_SUBSCRIBER,
						Type:   "offer",
						Sdp:    offer.SDP,
					},
				},
			},
		},
	)
}
