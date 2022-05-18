// Package rtc consists of a RTC to fetch tracks from other SFU
package rtc

import (
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
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
	signaller rtc.RTC_SignalClient
	peer      UpPeerLocal
	uid       string
	sync.Mutex
}

func NewRTC(peer UpPeerLocal, signaller rtc.RTC_SignalClient) *RTC {
	return &RTC{
		signaller: signaller,
		peer:      peer,
		uid:       peer.peer.ID(),
	}
}

// Run start a rtc from remote session to local session
func (r *RTC) Run(remoteSid, localSid string) error {

	r.peer.OnICECandidate(func(c *webrtc.ICECandidateInit) {
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
	err := r.peer.Join(localSid)
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
