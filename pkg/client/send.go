package client

import (
	"encoding/json"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

func (sub *Subscriber) SendJoin(sid string, uid string, config map[string]string) {
	log.Infof("[C=>S] sid=%v", sid)
	err := sub.client.Send(
		&pb.Request{
			Payload: &pb.Request_Join{
				Join: &pb.JoinRequest{
					Sid:    sid,
					Uid:    uid,
					Config: config,
				},
			},
		},
	)
	if err != nil {
		log.Errorf("Cannot send join: %v", err)
	}
}

func (sub *Subscriber) SendTrickle(candidate *webrtc.ICECandidate, target pb.Target) {
	log.Debugf("[C=>S] candidate=%v target=%v", candidate, target)
	bytes, err := json.Marshal(candidate.ToJSON())
	if err != nil {
		log.Errorf("Cannot marshal candidate: %v", err)
		return
	}
	err = sub.client.Send(
		&pb.Request{
			Payload: &pb.Request_Trickle{
				Trickle: &pb.Trickle{
					Target: target,
					Init:   string(bytes),
				},
			},
		},
	)
	if err != nil {
		log.Errorf("Cannot send candidate: %v", err)
	}
}

func (sub *Subscriber) SendAnswer(sdp webrtc.SessionDescription) {
	log.Infof("[C=>S] sdp=%v", sdp)
	err := sub.client.Send(
		&pb.Request{
			Payload: &pb.Request_Description{
				Description: &pb.SessionDescription{
					Target: pb.Target_SUBSCRIBER,
					Type:   "answer",
					Sdp:    sdp.SDP,
				},
			},
		},
	)
	if err != nil {
		log.Errorf("Cannot send answer: %v", err)
	}
}
