package sfu

import (
	"encoding/json"
	"errors"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/pkg/islb"
	pb2 "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/protobuf/proto"
	"sync"
)

type Publisher struct {
	client    *Client
	session   *pb2.ClientNeededSession
	NeedTrack func(AddTrack func(track webrtc.TrackLocal) (*webrtc.RTPSender, error)) error

	reconnectExec *util.SingleExec

	recvCandidates []webrtc.ICECandidateInit
	recvCandMu     sync.Mutex
}

func (pub *Publisher) Name() string {
	return "sfu.Publisher"
}

func (pub *Publisher) Connect() {
	pub.client.Connect()
}

func (pub *Publisher) Close() {
	pub.client.Close()
}

func (pub *Publisher) Connected() bool {
	return pub.client.Connected()
}

func (pub *Publisher) refresh() {
	pub.reconnectExec.Do(pub.client.Reconnect)
}

func NewPublisher(node *islb.Node) *Publisher {
	pub := &Publisher{
		client:        NewClient(node),
		reconnectExec: util.NewSingleExec(),
	}
	var lastPeerConnection *webrtc.PeerConnection
	pub.client.OnReconnect(func() {
		if lastPeerConnection != nil {
			_ = lastPeerConnection.Close()
		}
		pc, err := pub.newPeerConnection()
		if err != nil {
			pub.refresh()
			return
		}
		pub.client.OnMsgRecv(pub.newMsgHandler(pc))
		err = pub.NeedTrack(pc.AddTrack)
		if err != nil {
			pub.refresh()
			return
		}
		go func(pc *webrtc.PeerConnection) { // 必须这样，不然要是一直出错的话会无限递归
			offer, err := pc.CreateOffer(nil)
			if err != nil {
				log.Errorf("Cannot CreateOffer: %+v", err)
				pub.refresh()
				return
			}
			err = pc.SetLocalDescription(offer)
			if err != nil {
				log.Errorf("Cannot SetLocalDescription: %+v", err)
				pub.refresh()
				return
			}
			err = pub.SendJoin(pub.session.Session, pub.session.User, map[string]string{
				"NoSubscribe": "true",
			}, &offer)
			if err != nil {
				log.Errorf("Cannot Join %s, %s: %+v", pub.session.Session, pub.session.User, err)
				pub.refresh()
				return
			}
		}(pc)
		lastPeerConnection = pc
	})
	return pub
}

func (pub *Publisher) SwitchNode(peerNID string, parameters map[string]interface{}) {
	pub.client.Switch(peerNID, parameters)
}

func (pub *Publisher) SwitchSession(session *pb2.ClientNeededSession) {
	pub.session = proto.Clone(session).(*pb2.ClientNeededSession)
	pub.client.Reconnect()
}

func (pub *Publisher) Switch(peerNID string, parameters map[string]interface{}, session *pb2.ClientNeededSession) {
	pub.session = proto.Clone(session).(*pb2.ClientNeededSession)
	pub.client.Switch(peerNID, parameters)
}

func (pub *Publisher) newMsgHandler(pc *webrtc.PeerConnection) func(*pb.Reply) {
	handler := pub.msgHandler(pc)
	return func(reply *pb.Reply) {
		if err := handler(reply); err != nil {
			pub.refresh()
		}
	}
}

func (pub *Publisher) msgHandler(pc *webrtc.PeerConnection) func(*pb.Reply) error {
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
			if payload.Join.Description.Type == "answer" {
				sdp := webrtc.SessionDescription{
					SDP:  payload.Join.Description.Sdp,
					Type: webrtc.SDPTypeAnswer,
				}
				log.Infof("[description] got answer call s.OnNegotiate sdp=%+v", sdp)
				err := pub.negotiate(pc, &c, sdp)
				if err != nil {
					return err
				}
			} else {
				log.Warnf("[description] got offer sdp=%+v", payload.Join.Description.Sdp)
			}
		case *pb.Reply_Description:
			if payload.Description.Type == "answer" {
				sdp := webrtc.SessionDescription{
					SDP:  payload.Description.Sdp,
					Type: webrtc.SDPTypeAnswer,
				}
				log.Infof("[description] got answer call s.OnNegotiate sdp=%+v", sdp)
				err := pub.negotiate(pc, &c, sdp)
				if err != nil {
					return err
				}
			} else {
				log.Warnf("[description] got offer sdp=%+v", payload.Description.Sdp)
			}
		case *pb.Reply_Trickle:
			var candidate webrtc.ICECandidateInit
			_ = json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			log.Infof("[trickle] type=%v candidate=%+v", payload.Trickle.Target, candidate)
			pub.trickle(pc, &c, candidate, payload.Trickle.Target)
		case *pb.Reply_TrackEvent:
			log.Warnf("[track event] TrackEvent=%+v", payload.TrackEvent)
		case *pb.Reply_Subscription:
			if !payload.Subscription.Success {
				log.Warnf("[subscription] failed error: %v", payload.Subscription.Error)
			}
			log.Warnf("[subscription] success")
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
