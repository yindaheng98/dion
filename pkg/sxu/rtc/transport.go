package rtc

import (
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
)

// ↓↓↓↓↓ Similar to: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/transport.go ↓↓↓↓↓

// Transport is pub/sub transport
type Transport struct {
	api            *webrtc.DataChannel
	rtc            *RTC
	pc             *UpPeerLocal
	role           Target
	SendCandidates []*webrtc.ICECandidate
	RecvCandidates []webrtc.ICECandidateInit
}

// NewTransport create a transport
func NewTransport(rtc *RTC, pc *UpPeerLocal) *Transport {
	t := &Transport{
		role: Target_SUBSCRIBER,
		rtc:  rtc,
	}
	t.pc = pc
	t.pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			// Gathering done
			log.Infof("gather candidate done")
			return
		}
		//append before join session success
		if t.pc.CurrentRemoteDescription() == nil {
			t.SendCandidates = append(t.SendCandidates, c)
		} else {
			for _, cand := range t.SendCandidates {
				err := t.rtc.SendTrickle(cand, Target_SUBSCRIBER)
				if err != nil {
					if t.rtc.OnError != nil {
						t.rtc.OnError(err)
					}
					return
				}
			}
			t.SendCandidates = []*webrtc.ICECandidate{}
			err := t.rtc.SendTrickle(c, Target_SUBSCRIBER)
			if err != nil {
				if t.rtc.OnError != nil {
					t.rtc.OnError(err)
				}
				return
			}
		}
	})
	return t
}

// ↑↑↑↑↑ Similar to: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/transport.go ↑↑↑↑↑
