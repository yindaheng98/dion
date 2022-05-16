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

// RTC TODO 需要改造为Door给WatchDog用
type RTC struct {
	peer      *UpPeerLocal
	signaller rtc.RTC_SignalClient

	sub        *Transport
	OnError    func(error)
	uid        string
	handleOnce sync.Once
	sync.Mutex

	SendCandidates []*webrtc.ICECandidate
	RecvCandidates []webrtc.ICECandidateInit

	ctx    context.Context
	cancel context.CancelFunc
}

func NewRTC(sfu *ion_sfu.SFU) *RTC {
	peer := &UpPeerLocal{PeerLocal: ion_sfu.NewPeer(sfu)}
	r := &RTC{
		peer: peer,
		uid:  peer.ID(),
	}
	return r
}

// Start start a rtc from remote session to local session
func (r *RTC) Start(remoteSid, localSid string, client rtc.RTCClient, Metadata metadata.MD) error {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = metadata.NewOutgoingContext(ctx, Metadata)
	signaller, err := client.Signal(ctx)
	if err != nil {
		cancel()
		return err
	}
	r.signaller = signaller
	r.ctx = ctx
	r.cancel = cancel

	err = r.peer.Join(localSid)
	if err != nil {
		cancel()
		return err
	}
	r.sub = NewTransport(r, r.peer)

	err = r.SendJoin(remoteSid, r.peer.ID())
	if err != nil {
		cancel()
		return err
	}
	return nil
}

// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓

// GetSubStats get sub stats
func (r *RTC) GetSubStats() webrtc.StatsReport {
	return r.sub.pc.GetStats()
}

// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑

// trickle receive candidate from sfu and add to pc
func (r *RTC) trickle(candidate webrtc.ICECandidateInit, target Target) error {
	log.Debugf("[S=>C] id=%v candidate=%v target=%v", r.uid, candidate, target)
	var t *Transport
	if target == Target_SUBSCRIBER {
		t = r.sub
	} else {
		// t = r.pub
		return nil
	}

	if t.pc.CurrentRemoteDescription() == nil {
		t.RecvCandidates = append(t.RecvCandidates, candidate)
	} else {
		err := t.pc.AddICECandidate(candidate)
		if err != nil {
			log.Errorf("id=%v err=%v", r.uid, err)
			return err
		}
	}
	return nil
}

// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓

// negotiate sub negotiate
func (r *RTC) negotiate(sdp webrtc.SessionDescription) error {
	log.Debugf("[S=>C] id=%v Negotiate sdp=%v", r.uid, sdp)
	// 1.sub set remote sdp
	err := r.sub.pc.SetRemoteDescription(sdp)
	if err != nil {
		log.Errorf("id=%v Negotiate r.sub.pc.SetRemoteDescription err=%v", r.uid, err)
		return err
	}

	// 2. safe to send candiate to sfu after join ok
	if len(r.sub.SendCandidates) > 0 {
		for _, cand := range r.sub.SendCandidates {
			log.Debugf("[C=>S] id=%v send sub.SendCandidates r.uid, r.rtc.trickle cand=%v", r.uid, cand)
			err := r.SendTrickle(cand, Target_SUBSCRIBER)
			if err != nil {
				return err
			}
		}
		r.sub.SendCandidates = []*webrtc.ICECandidate{}
	}

	// 3. safe to add candidate after SetRemoteDescription
	if len(r.sub.RecvCandidates) > 0 {
		for _, candidate := range r.sub.RecvCandidates {
			log.Debugf("id=%v r.sub.pc.AddICECandidate candidate=%v", r.uid, candidate)
			_ = r.sub.pc.AddICECandidate(candidate)
		}
		r.sub.RecvCandidates = []webrtc.ICECandidateInit{}
	}

	// 4. create answer after add ice candidate
	answer, err := r.sub.pc.CreateAnswer(nil)
	if err != nil {
		log.Errorf("id=%v err=%v", r.uid, err)
		return err
	}

	// 5. set local sdp(answer)
	err = r.sub.pc.SetLocalDescription(answer)
	if err != nil {
		log.Errorf("id=%v err=%v", r.uid, err)
		return err
	}

	// 6. send answer to sfu
	err = r.SendAnswer(answer)
	if err != nil {
		log.Errorf("id=%v err=%v", r.uid, err)
		return err
	}
	return err
}

// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑

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
	for _, t := range r.peer.Publisher().PublisherTracks() {
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
	r.cancel()
	return r.peer.Close()
}
