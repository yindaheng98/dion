package sfu

import (
	"encoding/json"
	"fmt"
	log "github.com/pion/ion-log"
	error_code "github.com/pion/ion/pkg/error"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

func (sub *Subscriber) SendJoin(sid string, uid string, config map[string]string) error {
	return sub.client.SendJoin(sid, uid, config, nil)
}

func (pub *Publisher) SendJoin(sid string, uid string, config map[string]string, offer *webrtc.SessionDescription) error {
	return pub.client.SendJoin(sid, uid, config, offer)
}

func (c *Client) SendJoin(sid string, uid string, config map[string]string, offer *webrtc.SessionDescription) error {
	log.Infof("[C=>S] sid=%v", sid)
	var desc *pb.SessionDescription
	if offer != nil {
		desc = &pb.SessionDescription{
			Target: pb.Target_PUBLISHER,
			Type:   "offer",
			Sdp:    offer.SDP,
		}
	}
	config["IsClient"] = "true" // 为了那个很不优雅的客户端判断方式
	return c.Send(
		&pb.Request{
			Payload: &pb.Request_Join{
				Join: &pb.JoinRequest{
					Sid:         sid,
					Uid:         uid,
					Config:      config,
					Description: desc,
				},
			},
		},
	)
}

func (s Signaller) SendJoinError(err *pb.Error) error {
	return s.SendJoinReply(&pb.JoinReply{
		Success: false,
		Error:   err,
	})
}
func (s Signaller) SendJoinSuccess(answer *webrtc.SessionDescription) error {
	return s.SendJoinReply(&pb.JoinReply{
		Success: true,
		Description: &pb.SessionDescription{
			Target: pb.Target_PUBLISHER,
			Sdp:    answer.SDP,
			Type:   answer.Type.String(),
		},
	})
}
func (s Signaller) SendJoinReply(reply *pb.JoinReply) error {
	return s.sig.Send(&pb.Reply{
		Payload: &pb.Reply_Join{
			Join: reply,
		},
	})
}

func (s Signaller) SendError(code int32, reason string) error {
	return s.sig.Send(&pb.Reply{
		Payload: &pb.Reply_Error{
			Error: &pb.Error{
				Code:   code,
				Reason: reason,
			},
		},
	})
}

func (s Signaller) SendTrickle(candidate *webrtc.ICECandidateInit, target int) error {
	log.Debugf("[S=>C] peer.OnIceCandidate: target = %v, candidate = %v", target, candidate.Candidate)
	bytes, err := json.Marshal(candidate)
	if err != nil {
		log.Errorf("IceCandidate Marshal error: %v", err)
		return s.SendError(int32(error_code.InternalError), fmt.Sprintf("unmarshal ice candidate error:  %v", err))
	}
	return s.sig.Send(&pb.Reply{
		Payload: &pb.Reply_Trickle{
			Trickle: &pb.Trickle{
				Init:   string(bytes),
				Target: pb.Target(target),
			},
		},
	})
}

func (sub *Subscriber) SendTrickle(candidate *webrtc.ICECandidate) error {
	return sub.client.SendTrickle(candidate, pb.Target_SUBSCRIBER)
}

func (pub *Publisher) SendTrickle(candidate *webrtc.ICECandidate) error {
	return pub.client.SendTrickle(candidate, pb.Target_PUBLISHER)
}

func (c *Client) SendTrickle(candidate *webrtc.ICECandidate, target pb.Target) error {
	log.Debugf("[C=>S] candidate=%v target=%v", candidate, target)
	bytes, err := json.Marshal(candidate.ToJSON())
	if err != nil {
		log.Errorf("Cannot marshal candidate: %v", err)
		return err
	}
	return c.Send(
		&pb.Request{
			Payload: &pb.Request_Trickle{
				Trickle: &pb.Trickle{
					Target: target,
					Init:   string(bytes),
				},
			},
		},
	)
}

func (sub *Subscriber) SendAnswer(sdp webrtc.SessionDescription) error {
	log.Infof("[C=>S] sdp=%v", sdp)
	return sub.client.Send(
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
}

func (s Signaller) SendOffer(o *webrtc.SessionDescription) error {
	log.Debugf("[S=>C] peer.OnOffer: %v", o.SDP)
	return s.sig.Send(&pb.Reply{
		Payload: &pb.Reply_Description{
			Description: &pb.SessionDescription{
				Target: pb.Target_SUBSCRIBER,
				Sdp:    o.SDP,
				Type:   o.Type.String(),
			},
		},
	})
}
