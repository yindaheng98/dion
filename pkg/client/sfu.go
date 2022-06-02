package client

import (
	"encoding/json"
	"errors"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/pkg/sfu"
	pb2 "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"sync"
)

type Subscriber struct {
	client  *sfu.Client
	session *pb2.ClientNeededSession
	OnTrack func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)

	reconnectExec *util.SingleExec

	recvCandidates []webrtc.ICECandidateInit
	recvCandMu     sync.Mutex
}

func (sub *Subscriber) refresh() {
	sub.reconnectExec.Do(sub.client.Reconnect)
}

func NewSubscriber(node *ion.Node) *Subscriber {
	sub := &Subscriber{
		client:        sfu.NewClient(node),
		reconnectExec: util.NewSingleExec(),
	}
	var lastPeerConnection *webrtc.PeerConnection
	sub.client.OnReconnect(func() {
		if lastPeerConnection != nil {
			_ = lastPeerConnection.Close()
		}
		pc, err := sub.newPeerConnection()
		if err != nil {
			sub.refresh()
			return
		}
		sub.client.OnMsgRecv(sub.newMsgHandler(pc))
		go func() { // 必须这样，不然要是一直出错的话会无限递归
			err := sub.SendJoin(sub.session.Session, sub.session.User, map[string]string{})
			if err != nil {
				log.Errorf("Cannot Join %s, %s: %+v", sub.session.Session, sub.session.User, err)
				sub.refresh()
				return
			}
		}()
		lastPeerConnection = pc
	})
	return sub
}

func (sub *Subscriber) newMsgHandler(pc *webrtc.PeerConnection) func(*pb.Reply) {
	handler := sub.msgHandler(pc)
	return func(reply *pb.Reply) {
		if err := handler; err != nil {
			sub.refresh()
		}
	}
}

func (sub *Subscriber) msgHandler(pc *webrtc.PeerConnection) func(*pb.Reply) error {
	c := candidates{}
	return func(reply *pb.Reply) error {
		switch payload := reply.Payload.(type) {
		case *pb.Reply_Join:
			success := payload.Join.Success
			if !success {
				log.Errorf("[join] failed error: %v", payload.Join.Error.String())
				return errors.New(payload.Join.Error.String())
			}
			log.Infof("[join] success")
		case *pb.Reply_Description:
			if payload.Description.Type == "offer" {
				sdp := webrtc.SessionDescription{
					SDP:  payload.Description.Sdp,
					Type: webrtc.SDPTypeOffer,
				}
				log.Infof("[description] got offer call s.OnNegotiate sdp=%+v", sdp)
				err := sub.negotiate(pc, &c, sdp)
				if err != nil {
					return err
				}
			} else {
				log.Warnf("[description] got answer sdp=%+v", payload.Description.Sdp)
			}
		case *pb.Reply_Trickle:
			var candidate webrtc.ICECandidateInit
			_ = json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			log.Infof("[trickle] type=%v candidate=%+v", payload.Trickle.Target, candidate)
			sub.trickle(pc, &c, candidate, payload.Trickle.Target)
		case *pb.Reply_TrackEvent:
			log.Warnf("[track event] TrackEvent=%+v", payload.TrackEvent)
		case *pb.Reply_Subscription:
			if !payload.Subscription.Success {
				log.Errorf("[subscription] failed error: %v", payload.Subscription.Error)
				return errors.New(payload.Subscription.Error.String())
			}
			log.Infof("[subscription] success")
		case *pb.Reply_Error:
			log.Errorf("Request error: %v", payload.Error)
			return errors.New(payload.Error.String())
		default:
			log.Errorf("Unknown RTC type!!!!%v", payload)
			return errors.New("Unknown RTC type!!!! ")
		}
		return nil
	}

}
