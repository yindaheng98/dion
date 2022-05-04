package subscriber

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

type Subscriber struct {
	peer      *UpPeerLocal
	signaller rtc.RTC_SignalClient

	sub *Transport
	uid string
}

func NewSubscriber(peer *ion_sfu.PeerLocal, signaller rtc.RTC_SignalClient) *Subscriber {
	return &Subscriber{
		peer:      &UpPeerLocal{PeerLocal: peer},
		signaller: signaller,

		uid: peer.ID(),
	}
}

// Join join a remote session
func (r *Subscriber) Join(remoteSid, localSid string) error {
	r.peer.OnOffer = func(offer *webrtc.SessionDescription) {
		config := map[string]string{
			"NoPublish":       "true",
			"NoSubscribe":     "false",
			"NoAutoSubscribe": "false",
		}
		if err := r.SendJoin(remoteSid, r.uid, *offer, config); err != nil {
			log.Errorf("[Remote %v -> Local %v] error sending join request: %v", remoteSid, localSid, err)
		}
	}
	return r.peer.Join(localSid)
}

// GetSubStats get sub stats
func (r *Subscriber) GetSubStats() webrtc.StatsReport {
	return r.sub.pc.GetStats()
}

func (r *Subscriber) SendJoin(sid string, uid string, offer webrtc.SessionDescription, config map[string]string) error {
	// TODO
	return nil
}
