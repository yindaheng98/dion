package rtc

import (
	"encoding/json"
	"errors"
	"github.com/bep/debounce"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"sync"
	"time"
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
	peer      *UpPeerLocal
	signaller rtc.RTC_SignalClient

	sub        *Transport
	OnError    func(error)
	uid        string
	handleOnce sync.Once
	sync.Mutex

	SendCandidates []*webrtc.ICECandidate
	RecvCandidates []webrtc.ICECandidateInit

	tracksInfo []*rtc.TrackInfo
}

func NewRTC(peer *UpPeerLocal, signaller rtc.RTC_SignalClient) *RTC {
	r := &RTC{
		peer:      peer,
		signaller: signaller,

		uid: peer.ID(),
	}
	r.sub = NewTransport(r, r.peer)
	return r
}

// Start start a rtc from remote session to local session
func (r *RTC) Start(remoteSid, localSid string) error {
	err := r.peer.Join(localSid)
	if err != nil {
		return err
	}

	uid := r.uid
	publisher := r.sub.pc.Publisher()

	// ↓↓↓↓↓ Copy from: https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/service.go#L260 ↓↓↓↓↓
	var once sync.Once
	publisher.OnPublisherTrack(func(pt ion_sfu.PublisherTrack) {
		log.Debugf("[S=>C] OnPublisherTrack: \nKind %v, \nUid: %v,  \nMsid: %v,\nTrackID: %v", pt.Track.Kind(), uid, pt.Track.Msid(), pt.Track.ID())

		once.Do(func() {
			debounced := debounce.New(800 * time.Millisecond)
			debounced(func() {
				var peerTracks []*rtc.TrackInfo
				pubTracks := publisher.PublisherTracks()
				if len(pubTracks) == 0 {
					return
				}

				for _, pubTrack := range publisher.PublisherTracks() {
					peerTracks = append(peerTracks, &rtc.TrackInfo{
						Id:       pubTrack.Track.ID(),
						Kind:     pubTrack.Track.Kind().String(),
						StreamId: pubTrack.Track.StreamID(),
						Muted:    false,
						Layer:    pubTrack.Track.RID(),
					})
				}

				// broadcast the existing tracks in the session
				r.tracksInfo = append(r.tracksInfo, peerTracks...)
				log.Infof("[S=>C] BroadcastTrackEvent existing track %v, state = ADD", peerTracks)
				/*
					s.BroadcastTrackEvent(uid, peerTracks, rtc.TrackEvent_ADD)
					if err != nil {
						log.Errorf("signal send error: %v", err)
					}
				*/
			})
		})
	})
	// ↑↑↑↑↑ Copy from: https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/service.go#L260 ↑↑↑↑↑

	err = r.SendJoin(remoteSid, r.peer.ID())
	if err != nil {
		return err
	}
	return nil
}

// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓

// GetSubStats get sub stats
func (r *RTC) GetSubStats() webrtc.StatsReport {
	return r.sub.pc.GetStats()
}

// trickle receive candidate from sfu and add to pc
func (r *RTC) trickle(candidate webrtc.ICECandidateInit, target Target) {
	log.Debugf("[S=>C] id=%v candidate=%v target=%v", r.uid, candidate, target)
	var t *Transport
	if target == Target_SUBSCRIBER {
		t = r.sub
	} else {
		// t = r.pub
		return
	}

	if t.pc.CurrentRemoteDescription() == nil {
		t.RecvCandidates = append(t.RecvCandidates, candidate)
	} else {
		err := t.pc.AddICECandidate(candidate)
		if err != nil {
			log.Errorf("id=%v err=%v", r.uid, err)
		}
	}

}

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
			r.SendTrickle(cand, Target_SUBSCRIBER)
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

func (r *RTC) onSingalHandleOnce() {
	// onSingalHandle is wrapped in a once and only started after another public
	// method is called to ensure the user has the opportunity to register handlers
	r.handleOnce.Do(func() {
		err := r.onSingalHandle()
		if r.OnError != nil {
			r.OnError(err)
		}
	})
}

func (r *RTC) onSingalHandle() error {
	for {
		//only one goroutine for recving from stream, no need to lock
		stream, err := r.signaller.Recv()
		if err != nil {
			if err == io.EOF {
				log.Infof("[%v] WebRTC Transport Closed", r.uid)
				if err := r.signaller.CloseSend(); err != nil {
					log.Errorf("[%v] error sending close: %s", r.uid, err)
				}
				return err
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				if err := r.signaller.CloseSend(); err != nil {
					log.Errorf("[%v] error sending close: %s", r.uid, err)
				}
				return err
			}

			log.Errorf("[%v] Error receiving RTC response: %v", r.uid, err)
			if r.OnError != nil {
				r.OnError(err)
			}
			return err
		}

		switch payload := stream.Payload.(type) {
		case *rtc.Reply_Join:
			success := payload.Join.Success
			err := errors.New(payload.Join.Error.String())

			if !success {
				log.Errorf("[%v] [join] failed error: %v", r.uid, err)
				return err
			}
			log.Infof("[%v] [join] success", r.uid)
			/*
				log.Infof("payload.Reply.Description=%v", payload.Join.Description)
				sdp := webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  payload.Join.Description.Sdp,
				}

				if err = r.setRemoteSDP(sdp); err != nil {
					log.Errorf("[%v] [join] error %s", r.uid, err)
					return err
				}
			*/
		case *rtc.Reply_Description:
			var sdpType webrtc.SDPType
			if payload.Description.Type == "offer" {
				sdpType = webrtc.SDPTypeOffer
			} else {
				sdpType = webrtc.SDPTypeAnswer
			}
			sdp := webrtc.SessionDescription{
				SDP:  payload.Description.Sdp,
				Type: sdpType,
			}
			if sdp.Type == webrtc.SDPTypeOffer {
				log.Infof("[%v] [description] got offer call s.OnNegotiate sdp=%+v", r.uid, sdp)
				err := r.negotiate(sdp)
				if err != nil {
					log.Errorf("error: %v", err)
				}
			} else if sdp.Type == webrtc.SDPTypeAnswer {
				// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑
				log.Infof("[%v] [description] got answer call sdp=%+v, but i do not have a publisher", r.uid, sdp)
			}
			// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓
		case *rtc.Reply_Trickle:
			var candidate webrtc.ICECandidateInit
			_ = json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			log.Infof("[%v] [trickle] type=%v candidate=%v", r.uid, payload.Trickle.Target, candidate)
			r.trickle(candidate, Target(payload.Trickle.Target))
			/*
						case *rtc.Reply_TrackEvent:
							if r.OnTrackEvent == nil {
								log.Errorf("s.OnTrackEvent == nil")
								continue
							}
							var TrackInfos []*TrackInfo
							for _, v := range payload.TrackEvent.Tracks {
								TrackInfos = append(TrackInfos, &TrackInfo{
									Id:        v.Id,
									Kind:      v.Kind,
									Muted:     v.Muted,
									Type:      MediaType(v.Type),
									StreamId:  v.StreamId,
									Label:     v.Label,
									Width:     v.Width,
									Height:    v.Height,
									FrameRate: v.FrameRate,
									Layer:     v.Layer,
								})
							}
							trackEvent := TrackEvent{
								State:  TrackEvent_State(payload.TrackEvent.State),
								Uid:    payload.TrackEvent.Uid,
								Tracks: TrackInfos,
							}

							log.Infof("s.OnTrackEvent trackEvent=%+v", trackEvent)
							r.OnTrackEvent(trackEvent)
				case *rtc.Reply_Subscription:
					if !payload.Subscription.Success {
						log.Errorf("suscription error: %v", payload.Subscription.Error)
					}
			*/
		case *rtc.Reply_Error:
			log.Errorf("Request error: %v", payload.Error)
		default:
			log.Errorf("Unknown RTC type!!!!%v", payload)
		}
	}
}

// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑

func (r *RTC) SendJoin(sid string, uid string /*offer webrtc.SessionDescription, config map[string]string*/) error {
	// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓
	log.Infof("[C=>S] [%v] sid=%v", r.uid, sid)
	go r.onSingalHandleOnce()
	r.Lock()
	err := r.signaller.Send(
		&rtc.Request{
			Payload: &rtc.Request_Join{
				Join: &rtc.JoinRequest{
					Sid: sid,
					Uid: uid,
					// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑
					Config: map[string]string{
						"NoPublish":       "true",
						"NoSubscribe":     "false",
						"NoAutoSubscribe": "false",
					},
					// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓
					/*
						Description: &rtc.SessionDescription{
							Target: rtc.Target_PUBLISHER,
							Type:   "offer",
							Sdp:    offer.SDP,
						},
					*/
				},
			},
		},
	)
	r.Unlock()
	if err != nil {
		log.Errorf("[C=>S] [%v] err=%v", r.uid, err)
	}
	return err
}

func (r *RTC) SendTrickle(candidate *webrtc.ICECandidate, target Target) {
	log.Debugf("[C=>S] [%v] candidate=%v target=%v", r.uid, candidate, target)
	bytes, err := json.Marshal(candidate.ToJSON())
	if err != nil {
		log.Errorf("error: %v", err)
		return
	}
	go r.onSingalHandleOnce()
	r.Lock()
	err = r.signaller.Send(
		&rtc.Request{
			Payload: &rtc.Request_Trickle{
				Trickle: &rtc.Trickle{
					Target: rtc.Target(target),
					Init:   string(bytes),
				},
			},
		},
	)
	r.Unlock()
	if err != nil {
		log.Errorf("[%v] err=%v", r.uid, err)
	}
}

func (r *RTC) SendAnswer(sdp webrtc.SessionDescription) error {
	log.Infof("[C=>S] [%v] sdp=%v", r.uid, sdp)
	r.Lock()
	err := r.signaller.Send(
		&rtc.Request{
			Payload: &rtc.Request_Description{
				Description: &rtc.SessionDescription{
					Target: rtc.Target_SUBSCRIBER,
					Type:   "answer",
					Sdp:    sdp.SDP,
				},
			},
		},
	)
	r.Unlock()
	if err != nil {
		log.Errorf("[%v] err=%v", r.uid, err)
		return err
	}
	return nil
}

// Subscribe to tracks
func (r *RTC) Subscribe(trackInfos []*Subscription) error {
	if len(trackInfos) == 0 {
		return errors.New("track id is empty")
	}
	var infos []*rtc.Subscription
	for _, t := range trackInfos {
		infos = append(infos, &rtc.Subscription{
			TrackId:   t.TrackId,
			Mute:      t.Mute,
			Subscribe: t.Subscribe,
			Layer:     t.Layer,
		})
	}

	log.Infof("[C=>S] infos: %v", infos)
	err := r.signaller.Send(
		&rtc.Request{
			Payload: &rtc.Request_Subscription{
				Subscription: &rtc.SubscriptionRequest{
					Subscriptions: infos,
				},
			},
		},
	)
	return err
}

// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑

// Close stop all track
func (r *RTC) Close() error {
	peer := r.peer
	session := peer.Session()
	if session != nil {
		log.Infof("[S=>C] close: sid => %v, uid => %v", peer.Session().ID(), peer.ID())
		// Send Request_Subscribe to unsubscribe all tracks
		subs := make([]*Subscription, len(r.tracksInfo))
		for i, upTrack := range r.tracksInfo {
			subs[i] = &Subscription{
				TrackId:   upTrack.Id,
				Mute:      upTrack.Muted,
				Subscribe: false,
				Layer:     upTrack.Layer,
			}
		}
		return r.Subscribe(subs)
	}
	return nil
}
