package rtc

import (
	"github.com/pion/interceptor"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

// UpPeerLocal is a local peer that only have up tracks (tracks from other nodes)
type UpPeerLocal struct {
	peer  *ion_sfu.PeerLocal
	PubIr *interceptor.Registry
}

func NewUpPeerLocal(peer *ion_sfu.PeerLocal) UpPeerLocal {
	return UpPeerLocal{peer: peer}
}

// Join the up track peer join a session, with the option NoPublish = false and NoSubscribe = true
func (p *UpPeerLocal) Join(sid string) error {
	return p.peer.JoinWithInterceptorRegistry(sid, "", nil, p.PubIr, ion_sfu.JoinConfig{
		NoPublish:   false,
		NoSubscribe: true,
	})
}

func (p *UpPeerLocal) Answer(desc webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	return p.peer.Answer(desc)
}

func (p *UpPeerLocal) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	return p.peer.Trickle(candidate, int(rtc.Target_PUBLISHER))
}

/*
// Do not direct edit PeerConnection
func (p *UpPeerLocal) PeerConnection() *webrtc.PeerConnection {
	return p.Publisher().PeerConnection()
}
*/

func (p *UpPeerLocal) OnICECandidate(f func(c *webrtc.ICECandidateInit)) {
	p.peer.OnIceCandidate = func(init *webrtc.ICECandidateInit, target int) {
		if rtc.Target(target) == rtc.Target_PUBLISHER {
			f(init)
		}
	}
}

func (p *UpPeerLocal) GetStats() webrtc.StatsReport {
	return p.peer.Publisher().PeerConnection().GetStats()
}

func (p *UpPeerLocal) Close() error {
	return p.peer.Close()
}
