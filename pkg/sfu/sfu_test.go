package sfu

import (
	"context"
	"encoding/json"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"testing"

	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	"github.com/pion/ion/proto/rtc"
	pb "github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
)

var (
	conf = Config{}

	nid = "sfu-01"
)

func load_sfu() *SFU {
	confFile := "D:\\Documents\\MyPrograms\\dion\\pkg\\sxu\\sfu.toml"
	s := NewSFU(nid)
	err := conf.Load(confFile)
	if err != nil {
		panic(err)
	}
	isfu := ion_sfu.NewSFU(conf.Config)
	ss := NewSFUService(isfu)

	err = s.Start(conf.Common, ss)
	if err != nil {
		panic(err)
	}
	return s
}

func connect_sfu() rtc.RTC_SignalClient {
	opts := []nats.Option{nats.Name("nats-grpc echo client")}
	// Connect to the NATS server.
	nc, err := nats.Connect(conf.Nats.URL, opts...)
	if err != nil {
		panic(err)
	}

	ncli := rpc.NewClient(nc, nid, "unkown")
	cli := pb.NewRTCClient(ncli)

	stream, err := cli.Signal(context.Background())
	if err != nil {
		panic(err)
	}
	return stream
}

func init_pc(stream rtc.RTC_SignalClient, t *testing.T) *webrtc.PeerConnection {
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

func TestPub(t *testing.T) {
	s := load_sfu()
	stream := connect_sfu()
	pub := init_pc(stream, t)
	_, err := pub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}
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

	var candidates []webrtc.ICECandidateInit
	for {
		reply, err := stream.Recv()
		if err != nil {
			t.Errorf("Signal: err %s", err)
			break
		}
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
	}

	s.Close()
}

func TestSub(t *testing.T) {
	s := load_sfu()
	stream := connect_sfu()
	sub := init_pc(stream, t)
	_, err := sub.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		panic(err)
	}

	err = stream.Send(&pb.Request{
		Payload: &pb.Request_Join{
			Join: &pb.JoinRequest{
				Sid: "room1",
				Config: map[string]string{
					"NoPublish": "true",
				},
			},
		},
	})

	if err != nil {
		panic(err)
	}

	var candidates []webrtc.ICECandidateInit
	for {
		reply, err := stream.Recv()
		if err != nil {
			t.Errorf("Signal: err %s", err)
			break
		}
		t.Logf("\nReply: reply %v\n", reply)

		switch payload := reply.Payload.(type) {
		case *pb.Reply_Join:
			var sdpType webrtc.SDPType
			if payload.Join.Description.Type == "offer" {
				sdpType = webrtc.SDPTypeOffer
				sdp := webrtc.SessionDescription{
					SDP:  payload.Join.Description.Sdp,
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
	}

	s.Close()
}
