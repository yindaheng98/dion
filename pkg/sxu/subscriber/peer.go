package subscriber

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
)

// UpPeerLocal is a local peer that only have up tracks (tracks from other nodes)
type UpPeerLocal struct {
	ion_sfu.PeerLocal
}

// Join the up track peer join a session, with the option NoPublish = false and NoSubscribe = true
func (p *UpPeerLocal) Join(sid string) error {
	return p.PeerLocal.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:   false,
		NoSubscribe: true,
	})
}

// CreateOffer the up track peer Create an Offer for Publisher after join
func (p *UpPeerLocal) CreateOffer() (webrtc.SessionDescription, error) {
	offer, err := p.Publisher().PeerConnection().CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	if err := p.Publisher().PeerConnection().SetLocalDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	return offer, nil
}

func (p *UpPeerLocal) SetRemoteDescription(desc webrtc.SessionDescription) error {
	return p.Publisher().PeerConnection().SetRemoteDescription(desc)
}

func (p *UpPeerLocal) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	return p.Trickle(candidate, 1)
}
