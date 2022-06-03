package sfu

import (
	"encoding/json"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/yindaheng98/dion/pkg/islb"
	"testing"
	"time"

	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

func connect_sfu_1() *Client {
	node := islb.NewNode("sfu-client-test")
	err := node.Start(conf.Nats.URL)
	if err != nil {
		panic(err)
	}
	//重要！！！必须开启了Watch才能获取到其他节点.
	go func() {
		err := node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()
	return NewClient(&node)
}

func init_pc_1(stream *Client, t *testing.T) *webrtc.PeerConnection {
	me := webrtc.MediaEngine{}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		t.Logf("ICEConnectionState %v", state.String())
	})

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		t.Logf("OnICECandidate %v", candidate)
		bytes, err := json.Marshal(candidate)
		if err != nil {
			panic(err)
		}
		err = stream.Send(&pb.Request{
			Payload: &pb.Request_Trickle{
				Trickle: &pb.Trickle{
					Target: pb.Target_PUBLISHER,
					Init:   string(bytes),
				},
			},
		})
		if err != nil {
			panic(err)
		}
	})
	return pc
}

func TestClientPub(t *testing.T) {
	s := load_sfu()
	stream := connect_sfu_1()
	pub := init_pc_1(stream, t)
	_, err := pub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}
	sub := init_pc_1(stream, t)
	_, err = sub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}

	var candidates []webrtc.ICECandidateInit
	stream.OnMsgRecv(func(reply *pb.Reply) {
		t.Logf("\nReply: reply %v\n", reply)
		switch payload := reply.Payload.(type) {
		case *pb.Reply_Join:
			var sdpType webrtc.SDPType
			if payload.Join.Description.Type == "offer" {
				sdpType = webrtc.SDPTypeOffer
			} else {
				sdpType = webrtc.SDPTypeAnswer
				sdp := webrtc.SessionDescription{
					SDP:  payload.Join.Description.Sdp,
					Type: sdpType,
				}
				err = pub.SetRemoteDescription(sdp)
				if err != nil {
					panic(err)
				}
			}
		case *pb.Reply_Description:
			var sdpType webrtc.SDPType
			if payload.Description.Type == "offer" {
				sdpType = webrtc.SDPTypeOffer
				sdp := webrtc.SessionDescription{
					SDP:  payload.Description.Sdp,
					Type: sdpType,
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
				err = stream.Send(&pb.Request{
					Payload: &pb.Request_Description{
						Description: &pb.SessionDescription{
							Target: pb.Target_SUBSCRIBER,
							Type:   "answer",
							Sdp:    answer.SDP,
						},
					},
				})
				if err != nil {
					panic(err)
				}
			} else {
				sdpType = webrtc.SDPTypeAnswer
				sdp := webrtc.SessionDescription{
					SDP:  payload.Description.Sdp,
					Type: sdpType,
				}
				err = pub.SetRemoteDescription(sdp)
				if err != nil {
					panic(err)
				}
			}
		case *pb.Reply_Trickle:
			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			if err != nil {
				panic(err)
			}
			candidates = append(candidates, candidate)
			cs := candidates
			candidates = []webrtc.ICECandidateInit{}
			if pub.CurrentLocalDescription() != nil {
				for _, c := range cs {
					err = pub.AddICECandidate(c)
					if err != nil {
						panic(err)
					}
				}
			}
			//return
		}
	})
	stream.Connect()

	offer, err := pub.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	t.Logf("offer => %v", offer)
	err = stream.Send(&pb.Request{
		Payload: &pb.Request_Join{
			Join: &pb.JoinRequest{
				Sid: "room1",
				Description: &pb.SessionDescription{
					Target: pb.Target_PUBLISHER,
					Type:   offer.Type.String(),
					Sdp:    offer.SDP,
				},
				Config: map[string]string{
					"NoAutoSubscribe": "true",
				},
			},
		},
	})

	if err != nil {
		panic(err)
	}

	err = pub.SetLocalDescription(offer)

	if err != nil {
		panic(err)
	}
	stream.OnReconnect(func() {
		t.Logf("Haha! reconnecting")
	})
	for i := 0; i < 10; i++ {
		<-time.After(5 * time.Second)
		stream.Switch("unknown", map[string]interface{}{})
		<-time.After(5 * time.Second)
		stream.Switch("*", map[string]interface{}{})
	}
	<-time.After(10 * time.Second)
	stream.Close() // 这里断开连接并不会让对方断开连接。但是没关系，因为SFU可以从PeerConnection断开连接
	<-time.After(10 * time.Second)
	pub.Close()
	sub.Close()
	<-time.After(10 * time.Second)
	s.Close()
}

func TestClientSub(t *testing.T) {
	s := load_sfu()
	stream := connect_sfu_1()
	pub := init_pc_1(stream, t)
	_, err := pub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}
	sub := init_pc_1(stream, t)
	_, err = sub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}

	var candidates []webrtc.ICECandidateInit
	stream.OnMsgRecv(func(reply *pb.Reply) {
		t.Logf("\nReply: reply %v\n", reply)
		switch payload := reply.Payload.(type) {
		case *pb.Reply_Description:
			var sdpType webrtc.SDPType
			if payload.Description.Type == "offer" {
				sdpType = webrtc.SDPTypeOffer
				sdp := webrtc.SessionDescription{
					SDP:  payload.Description.Sdp,
					Type: sdpType,
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
				err = stream.Send(&pb.Request{
					Payload: &pb.Request_Description{
						Description: &pb.SessionDescription{
							Target: pb.Target_SUBSCRIBER,
							Type:   "answer",
							Sdp:    answer.SDP,
						},
					},
				})
				if err != nil {
					panic(err)
				}
			} else {
				sdpType = webrtc.SDPTypeAnswer
			}
		case *pb.Reply_Trickle:
			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			if err != nil {
				panic(err)
			}
			candidates = append(candidates, candidate)
			cs := candidates
			candidates = []webrtc.ICECandidateInit{}
			if sub.CurrentLocalDescription() != nil {
				for _, c := range cs {
					err = sub.AddICECandidate(c)
					if err != nil {
						panic(err)
					}
				}
			}
			//return
		}

	})
	stream.Connect()

	offer, err := pub.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	t.Logf("offer => %v", offer)
	err = stream.Send(&pb.Request{
		Payload: &pb.Request_Join{
			Join: &pb.JoinRequest{
				Sid: "room1",
				Description: &pb.SessionDescription{
					Target: pb.Target_PUBLISHER,
					Type:   offer.Type.String(),
					Sdp:    offer.SDP,
				},
				Config: map[string]string{
					// "NoPublish": "true", // TODO: 这段代码里只要加上这个就会SetRemoteDescription called with no ice-ufrag
				},
			},
		},
	})

	if err != nil {
		panic(err)
	}
	stream.OnReconnect(func() {
		t.Logf("Haha! reconnecting")
	})
	for i := 0; i < 10; i++ {
		<-time.After(5 * time.Second)
		stream.Switch("unknown", map[string]interface{}{})
		<-time.After(5 * time.Second)
		stream.Switch("*", map[string]interface{}{})
	}
	<-time.After(10 * time.Second)
	stream.Close()
	<-time.After(10 * time.Second)
	pub.Close()
	sub.Close()
	<-time.After(10 * time.Second)
	s.Close()
}
