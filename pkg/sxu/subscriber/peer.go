package subscriber

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
)

// UpPeerLocal is a local peer that only have up tracks (tracks from other nodes)
type UpPeerLocal struct {
	*ion_sfu.PeerLocal
}

// Join the up track peer join a session, with the option NoPublish = false and NoSubscribe = true
func (p *UpPeerLocal) Join(sid string) error {
	return p.PeerLocal.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:   false,
		NoSubscribe: true,
	})
}

func (p *UpPeerLocal) SetRemoteDescription(desc webrtc.SessionDescription) error {
	return p.Publisher().PeerConnection().SetRemoteDescription(desc)
}

func (p *UpPeerLocal) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	return p.Publisher().AddICECandidate(candidate)
}

/*
// Do not direct edit PeerConnection
func (p *UpPeerLocal) PeerConnection() *webrtc.PeerConnection {
	return p.Publisher().PeerConnection()
}
*/

func (p *UpPeerLocal) OnICECandidate(f func(c *webrtc.ICECandidate)) {
	p.Publisher().OnICECandidate(f)
}

func (p *UpPeerLocal) CurrentRemoteDescription() *webrtc.SessionDescription {
	return p.Publisher().PeerConnection().CurrentRemoteDescription()
}

func (p *UpPeerLocal) GetStats() webrtc.StatsReport {
	return p.Publisher().PeerConnection().GetStats()
}

func (p *UpPeerLocal) CreateAnswer(options *webrtc.AnswerOptions) (webrtc.SessionDescription, error) {
	return p.Publisher().PeerConnection().CreateAnswer(options)
}

func (p *UpPeerLocal) SetLocalDescription(sdp webrtc.SessionDescription) error {
	return p.Publisher().PeerConnection().SetLocalDescription(sdp)
}
