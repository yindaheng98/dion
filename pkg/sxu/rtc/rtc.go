package rtc

import (
	"context"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/grpc/metadata"
	"sync"
)

type Subscription struct {
	TrackId   string
	Mute      bool
	Subscribe bool
	Layer     string
}

type Target int32

const (
	Target_PUBLISHER  Target = 0
	Target_SUBSCRIBER Target = 1
)

type RTC struct {
	sfu *ion_sfu.SFU

	peer      UpPeerLocal
	signaller rtc.RTC_SignalClient
	uid       string
	sync.Mutex
}

func NewRTC(sfu *ion_sfu.SFU) *RTC {
	return &RTC{
		sfu: sfu,
	}
}

// Run start a rtc from remote session to local session
func (r *RTC) Run(remoteSid, localSid string, client rtc.RTCClient, Metadata metadata.MD) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize GRPC signaller
	ctx = metadata.NewOutgoingContext(ctx, Metadata)
	signaller, err := client.Signal(ctx)
	if err != nil {
		return err
	}
	defer func(signaller rtc.RTC_SignalClient) {
		if err := signaller.CloseSend(); err != nil {
			log.Errorf("Error when CloseSend: %+v", err)
		}
	}(signaller)
	r.signaller = signaller

	// Initialize UpPeerLocal
	peer := UpPeerLocal{peer: ion_sfu.NewPeer(r.sfu)}
	defer func(peer UpPeerLocal) {
		if err := peer.Close(); err != nil {
			log.Errorf("Error when Close PeerLocal: %+v", err)
		}
	}(peer)
	peer.OnICECandidate(func(c *webrtc.ICECandidateInit) {
		if c == nil {
			// Gathering done
			log.Infof("id=%s gather candidate done", r.uid)
			return
		}
		//append before join session success
		err := r.SendTrickleInit(*c, Target_SUBSCRIBER)
		if err != nil {
			return
		}
	})

	// Join a local session
	err = r.peer.Join(localSid)
	if err != nil {
		return err
	}

	// Join a remote session
	err = r.SendJoin(remoteSid, r.uid)
	if err != nil {
		return err
	}

	// Handle messages
	return r.onSingalHandle()
}

// GetSubStats get sub stats
func (r *RTC) GetSubStats() webrtc.StatsReport {
	return r.peer.GetStats()
}

// trickle receive candidate from sfu and add to pc
func (r *RTC) trickle(candidate webrtc.ICECandidateInit, target Target) error {
	log.Debugf("[S=>C] id=%v candidate=%v target=%v", r.uid, candidate, target)
	err := r.peer.AddICECandidate(candidate)
	if err != nil {
		log.Errorf("id=%v err=%v", r.uid, err)
		return err
	}
	return nil
}

// negotiate sub negotiate
func (r *RTC) negotiate(offer webrtc.SessionDescription) error {
	log.Debugf("[S=>C] id=%v Negotiate sdp=%v", r.uid, offer)

	answer, err := r.peer.Answer(offer)
	if err != nil {
		log.Errorf("id=%v Negotiate Answer err=%v", r.uid, err)
		return err
	}

	err = r.SendAnswer(*answer)
	if err != nil {
		log.Errorf("id=%v SendAnswer err=%v", r.uid, err)
		return err
	}
	return err
}

var Layers = map[pb.Subscription_Layer]string{
	pb.Subscription_Q: "q",
	pb.Subscription_H: "h",
	pb.Subscription_F: "f",
}

func (r *RTC) Update(tracks []*pb.Subscription) error {
	trackInfos := make([]*Subscription, len(tracks))
	for i, track := range tracks {
		trackInfos[i] = &Subscription{
			TrackId:   track.TrackId,
			Mute:      track.Mute,
			Subscribe: true,
			Layer:     Layers[track.Layer],
		}
	}
	return r.Subscribe(trackInfos)
}

func (r *RTC) IsSame(tracks []*pb.Subscription) bool {
	temp := map[string]*webrtc.TrackRemote{}
	for _, t := range r.peer.peer.Publisher().PublisherTracks() {
		temp[t.Track.ID()] = t.Track
	}
	for _, sub := range tracks {
		if t, ok := temp[sub.TrackId]; ok {
			if !TrackSame(sub, t) {
				return false
			}
		}
	}
	return true
}

// Close stop all track
func (r *RTC) Close() error {
	return r.signaller.CloseSend()
}
