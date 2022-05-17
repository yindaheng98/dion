package rtc

import (
	"encoding/json"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

func (r *RTC) SendJoin(sid string, uid string /*offer webrtc.SessionDescription, config map[string]string*/) error {
	// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓
	log.Infof("[C=>S] [%v] sid=%v", r.uid, sid)
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

// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑

func (r *RTC) SendTrickle(candidate *webrtc.ICECandidate, target Target) error {
	return r.SendTrickleInit(candidate.ToJSON(), target)
}

func (r *RTC) SendTrickleInit(candidate webrtc.ICECandidateInit, target Target) error {
	log.Debugf("[C=>S] [%v] candidate=%v target=%v", r.uid, candidate, target)
	bytes, err := json.Marshal(candidate)
	if err != nil {
		log.Errorf("error: %v", err)
		return err
	}
	// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓
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
	// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑
	return err
}

// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓

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
		return nil
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
