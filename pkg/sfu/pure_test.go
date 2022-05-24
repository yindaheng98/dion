package sfu

import (
	"encoding/json"
	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"testing"
	"time"
)

func init_pc_pure(t *testing.T, peer *ion_sfu.PeerLocal) *webrtc.PeerConnection {
	me := webrtc.MediaEngine{}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		t.Logf("ICEConnectionState %v", state.String())
	})

	pc.OnICECandidate(func(candidate0 *webrtc.ICECandidate) {
		if candidate0 == nil {
			return
		}
		t.Logf("OnICECandidate %v", candidate0)
		bytes, err := json.Marshal(candidate0)
		if err != nil {
			panic(err)
		}

		var candidate webrtc.ICECandidateInit
		err = json.Unmarshal(bytes, &candidate)
		if err != nil {
			panic(err)
		}
		go func(peer *ion_sfu.PeerLocal) {
			err = peer.Trickle(candidate, int(pb.Target_PUBLISHER))
			if err != nil {
				panic(err)
			}
		}(peer)
	})
	return pc
}

func TestSubPure(t *testing.T) {
	confFile := "D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml"
	err := conf.Load(confFile)
	if err != nil {
		panic(err)
	}
	isfu := ion_sfu.NewSFU(conf.Config)
	dc := isfu.NewDatachannel(ion_sfu.APIChannelLabel)
	dc.Use(datachannel.SubscriberAPI)

	peer := ion_sfu.NewPeer(isfu,
		ion_sfu.WithPubInterceptorRegistryFactoryBuilder(PubIRFBuilder{}),
		ion_sfu.WithSubInterceptorRegistryFactoryBuilder(PubIRFBuilder{}),
	)

	sub := init_pc_pure(t, peer)
	_, err = sub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}
	pub := init_pc_pure(t, peer)
	_, err = pub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}
	offer, err := pub.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	var candidates []webrtc.ICECandidateInit

	peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
		log.Debugf("[S=>C] peer.OnIceCandidate: target = %v, candidate = %v", target, candidate.Candidate)
		candidates = append(candidates, *candidate)
		cs := candidates
		candidates = []webrtc.ICECandidateInit{}
		go func(cs []webrtc.ICECandidateInit) {
			if sub.CurrentLocalDescription() != nil {
				for _, c := range cs {
					err = sub.AddICECandidate(c)
					if err != nil {
						panic(err)
					}
				}
			}
		}(cs)
	}

	// Notify user of new offer
	peer.OnOffer = func(o *webrtc.SessionDescription) {
		log.Debugf("[S=>C] peer.OnOffer: %v", o.SDP)
		sdp := webrtc.SessionDescription{
			SDP:  o.SDP,
			Type: webrtc.SDPTypeOffer,
		}
		err = sub.SetRemoteDescription(sdp)
		if err != nil {
			panic(err)
		}
		answer, err := sub.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}
		err = sub.SetLocalDescription(answer)
		if err != nil {
			panic(err)
		}
		go func(answer webrtc.SessionDescription) {
			desc := webrtc.SessionDescription{
				SDP:  answer.SDP,
				Type: webrtc.NewSDPType("answer"),
			}
			err = peer.SetRemoteDescription(desc)
			if err != nil {
				panic(err)
			}
		}(answer)
	}

	err = peer.Join("room1", "")

	desc := webrtc.SessionDescription{
		SDP:  offer.SDP,
		Type: webrtc.NewSDPType(offer.Type.String()),
	}

	log.Debugf("[C=>S] join.description: offer %v", desc.SDP)
	_, err = peer.Answer(desc)
	if err != nil {
		panic(err)
	}

	<-time.After(30 * time.Second)
}
