package bridge

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/util"
)

type PublisherFactory struct {
	sfu *ion_sfu.SFU
	SID string
}

func (p PublisherFactory) NewDoor() (util.Door, error) {
	peer := ion_sfu.NewPeer(p.sfu)
	me, err := getSubscriberMediaEngine()
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
	return Publisher{
		bridgePeer: bridgePeer{
			peer: peer,
			pc:   pc,
		},
		sid: p.SID,
	}, nil
}

type Publisher struct {
	bridgePeer
	sid string
}

func (p Publisher) Lock(OnBroken func(badGay error)) error {
	return p.publish(p.sid, OnBroken)
}

func (p bridgePeer) Repair() bool {
	return false
}

func NewPublisher(peer *ion_sfu.PeerLocal, pc *webrtc.PeerConnection) Publisher {
	return Publisher{bridgePeer: newPeer(peer, pc)}
}

// publish publish PeerConnection to PeerLocal.Subscriber
func (p bridgePeer) publish(sid string, OnBroken func(badGay error)) error {
	p.pc.OnNegotiationNeeded(func() {
		offer, err := p.pc.CreateOffer(nil)
		if err != nil {
			log.Errorf("Cannot CreateOffer in pc: %+v", err)
			OnBroken(err)
			return
		}
		err = p.pc.SetLocalDescription(offer)
		if err != nil {
			log.Errorf("Cannot SetLocalDescription to pc: %+v", err)
			OnBroken(err)
			return
		}
		answer, err := p.peer.Answer(offer)
		if err != nil {
			log.Errorf("Cannot create Answer in peer: %+v", err)
			OnBroken(err)
			return
		}
		err = p.pc.SetRemoteDescription(*answer)
		if err != nil {
			log.Errorf("Cannot SetRemoteDescription to pc: %+v", err)
			OnBroken(err)
			return
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

	candidateSetting(p.pc, p.peer, OnBroken, rtc.Target_PUBLISHER)

	return err
}

func (p Publisher) AddTrack(track webrtc.TrackLocal) (*webrtc.RTPSender, error) {
	addTrack, err := p.pc.AddTrack(track)
	if err != nil {
		return nil, err
	}
	return addTrack, nil
}
