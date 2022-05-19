package bridge

import (
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
)

type PublisherFactory struct {
	sfu *ion_sfu.SFU
}

func NewPublisherFactory(sfu *ion_sfu.SFU) PublisherFactory {
	return PublisherFactory{sfu: sfu}
}

func (p PublisherFactory) NewDoor() (util.UnblockedDoor, error) {
	me, err := getSubscriberMediaEngine()
	if err != nil {
		log.Errorf("Cannot getSubscriberMediaEngine for pc: %+v", err)
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(webrtc.SettingEngine{}))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Errorf("Cannot NewPeerConnection: %+v", err)
		return nil, err
	}
	return Publisher{BridgePeer: NewBridgePeer(ion_sfu.NewPeer(p.sfu), pc)}, nil
}

type Publisher struct {
	BridgePeer
}

func (p Publisher) Lock(sid util.Param, OnBroken func(badGay error)) error {
	return p.publish(string(sid.(SID)), OnBroken)
}

func (p Publisher) Repair(util.Param) error {
	return fmt.Errorf("Publisher cannot be repaired ")
}

func (p Publisher) Update(util.Param) error {
	return fmt.Errorf("Publisher cannot be updated ")
}

// publish publish PeerConnection to PeerLocal.Subscriber
func (p Publisher) publish(sid string, OnBroken func(error)) error {
	addCandidate := p.setOnIceCandidate(OnBroken, rtc.Target_PUBLISHER)
	p.pc.OnNegotiationNeeded(func() {
		if err := p.onNegotiationNeeded(); err != nil {
			OnBroken(err)
		}
	})

	err := p.peer.Join(sid, "", ion_sfu.JoinConfig{
		NoPublish:       false,
		NoSubscribe:     true,
		NoAutoSubscribe: false,
	})
	if err != nil {
		return err
	}

	addCandidate()

	return err
}
func (p Publisher) onNegotiationNeeded() error {
	log.Infof("Bridge need a Negotiation to publish a track to SFU session")
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		log.Errorf("Cannot CreateOffer in pc: %+v", err)
		return err
	}
	err = p.pc.SetLocalDescription(offer)
	if err != nil {
		log.Errorf("Cannot SetLocalDescription to pc: %+v", err)
		return err
	}
	answer, err := p.peer.Answer(offer)
	if err != nil {
		log.Errorf("Cannot create Answer in peer: %+v", err)
		return err
	}
	err = p.pc.SetRemoteDescription(*answer)
	if err != nil {
		log.Errorf("Cannot SetRemoteDescription to pc: %+v", err)
		return err
	}
	return nil
}
func (p Publisher) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	addTrack, err := p.pc.AddTrack(track)
	if err != nil {
		log.Errorf("Cannot AddTrack to pc: %+v", err)
		return nil, err
	}
	return addTrack, nil
}

func (p Publisher) RemoveTrack(sender *webrtc.RTPSender) error {
	if err := p.pc.RemoveTrack(sender); err != nil {
		return err
	}
	return nil
}
