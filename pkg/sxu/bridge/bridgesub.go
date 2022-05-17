package bridge

import (
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
)

type SubscriberFactory struct {
	sfu *ion_sfu.SFU
}

func NewSubscriberFactory(sfu *ion_sfu.SFU) SubscriberFactory {
	return SubscriberFactory{sfu: sfu}
}

func (s SubscriberFactory) NewDoor() (util.Door, error) {
	me, err := getPublisherMediaEngine()
	if err != nil {
		log.Errorf("Cannot getPublisherMediaEngine for pc: %+v", err)
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(webrtc.SettingEngine{}))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Errorf("Cannot NewPeerConnection: %+v", err)
		return nil, err
	}
	return Subscriber{BridgePeer: NewBridgePeer(ion_sfu.NewPeer(s.sfu), pc)}, nil
}

type Subscriber struct {
	BridgePeer
}

func (s Subscriber) Lock(sid util.Param, OnBroken func(badGay error)) error {
	return s.subscribe(string(sid.(SID)), OnBroken)
}

func (s Subscriber) Repair(util.Param) error {
	return fmt.Errorf("Subscriber cannot be repaired ")
}

func (s Subscriber) Update(util.Param) error {
	return fmt.Errorf("Subscriber cannot be updated ")
}

// subscribe subscribe PeerConnection to PeerLocal.Subscriber
func (s Subscriber) subscribe(sid string, OnBroken func(error)) error {
	addCandidate := s.setOnIceCandidate(OnBroken, rtc.Target_SUBSCRIBER)
	s.peer.OnOffer = func(offer *webrtc.SessionDescription) {
		log.Infof("Bridge get a new offer to subscribe a track from SFU session %s", sid)
		err := s.pc.SetRemoteDescription(*offer)
		if err != nil {
			log.Errorf("Cannot SetRemoteDescription to pc: %+v", err)
			OnBroken(err)
			return
		}
		addCandidate()
		answer, err := s.pc.CreateAnswer(nil)
		if err != nil {
			log.Errorf("Cannot CreateAnswer in pc: %+v", err)
			OnBroken(err)
			return
		}
		err = s.pc.SetLocalDescription(answer)
		if err != nil {
			log.Errorf("Cannot SetLocalDescription in pc: %+v", err)
			OnBroken(err)
			return
		}
		go func(answer webrtc.SessionDescription) {
			err = s.peer.SetRemoteDescription(answer)
			if err != nil {
				log.Errorf("Cannot SetRemoteDescription in BridgePeer: %+v", err)
				OnBroken(err)
				return
			}
		}(answer)
	}

	err := s.peer.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:       true,
		NoSubscribe:     false,
		NoAutoSubscribe: false,
	})
	if err != nil {
		return err
	}

	return err
}

func (s Subscriber) OnTrack(f func(*webrtc.TrackRemote, *webrtc.RTPReceiver)) {
	s.pc.OnTrack(f)
}
