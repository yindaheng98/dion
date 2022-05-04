package sfu

import (
	"context"
	"encoding/json"

	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/node/sfu"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/pkg/sxu"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/grpc"
)

type Stream struct {
	util.ForwardTrackItem
	ctx    context.Context
	cancel context.CancelFunc
}

// ForwardController controls the track forward in SFU
type ForwardController struct {
	*sfu.SFUService
	node      *ion.Node
	sfu       *ion_sfu.SFU
	grpcConns map[string]grpc.ClientConnInterface
	streams   map[string]Stream
}

func NewSFUController(SFU *ion_sfu.SFU) *ForwardController {
	return &ForwardController{sfu: SFU, SFUService: sfu.NewSFUServiceWithSFU(SFU)}
}

func (s *ForwardController) StartTransmit(trackInfo *pb.ForwardTrack) error {
	// TODO: transmit a track
	grpcConn, ok := s.grpcConns[trackInfo.Src.Nid]
	if !ok {
		var err error
		grpcConn, err = s.node.NewNatsRPCClient(sxu.ServiceSXU, trackInfo.Src.Nid, map[string]interface{}{})
		if err != nil {
			log.Errorf("Cannot make GRPC Connection: %+v", err)
			return err
		}
		s.grpcConns[trackInfo.Src.Nid] = grpcConn
	}
	signalClient := rtc.NewRTCClient(grpcConn)
	ctx, cancel := context.WithCancel(context.Background())
	signalStream, err := signalClient.Signal(ctx)
	if err != nil {
		log.Errorf("Cannot start GRPC Stream: %+v", err)
		cancel()
		return err
	}
	stream := Stream{
		ForwardTrackItem: util.ForwardTrackItem{Track: trackInfo},
		ctx:              ctx,
		cancel:           cancel,
	}
	s.streams[stream.ForwardTrackItem.Key()] = stream
	go func(cancel context.CancelFunc) {
		defer cancel()
		err := s.Signal(signalStream, stream)
		if err != nil {
			log.Errorf("Error when signalling: %+v", err)
		}
	}(cancel)
	return nil
}

func (s *ForwardController) Signal(sig rtc.RTC_SignalClient, info Stream) error {
	// TODO: Signal func in client side, send request, receive reply (Signal func in server side receive request and send reply)

	peer := ion_sfu.NewPeer(s.sfu)

	// ↓↓↓↓↓↓ Copy fom https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/service.go#L144 ↓↓↓↓↓↓
	peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
		// Send ICECandidate
		bytes, err := json.Marshal(candidate)
		if err != nil {
			log.Errorf("OnIceCandidate error: %v", err)
		}
		err = sig.Send(&rtc.Request{
			Payload: &rtc.Request_Trickle{
				Trickle: &rtc.Trickle{
					Init:   string(bytes),
					Target: rtc.Target(target),
				},
			},
		})
		if err != nil {
			log.Errorf("OnIceCandidate send error: %v", err)
		}
	}

	peer.OnOffer = func(o *webrtc.SessionDescription) {
		log.Debugf("[S=>C] peer.OnOffer: %v", o.SDP)
		// Then all the last just send SDP
		err := sig.Send(&rtc.Request{
			Payload: &rtc.Request_Description{
				Description: &rtc.SessionDescription{
					Target: rtc.Target(rtc.Target_SUBSCRIBER),
					Sdp:    o.SDP,
					Type:   o.Type.String(),
				},
			},
		})
		if err != nil {
			log.Errorf("negotiation error: %v", err)
		}
	}
	// ↑↑↑↑↑↑ Copy fom https://github.com/pion/ion/blob/65dbd12eaad0f0e0a019b4d8ee80742930bcdc28/pkg/node/sfu/service.go#L164 ↑↑↑↑↑↑

	err := peer.Join(info.Track.LocalSessionId, "")
	if err != nil {
		log.Errorf("Cannot create local peer: %+v", err)
		return err
	}

	offer, err := peer.Subscriber().CreateOffer()
	if err != nil {
		log.Errorf("Cannot create offer: %+v", err)
		return err
	}

	// Send JoinRequest at first
	err = sig.Send(&rtc.Request{
		Payload: &rtc.Request_Join{
			Join: &rtc.JoinRequest{
				Sid:    info.Track.RemoteSessionId,
				Uid:    peer.ID(),
				Config: map[string]string{"NoPublish": "true"},
				Description: &rtc.SessionDescription{
					Target: rtc.Target(rtc.Target_SUBSCRIBER),
					Sdp:    offer.SDP,
					Type:   offer.Type.String(),
				},
			},
		},
	})
	if err != nil {
		log.Errorf("negotiation error: %v", err)
	}
	// TODO: 直接hack ion-sdk-go，把RTC里面对PeerConnection的操作替换成对ion-sfu的操作
	return nil
}

func (s *ForwardController) StopTransmit(trackInfo *pb.ForwardTrack) error {
	// TODO: stop a track transition
	return nil
}

func (s *ForwardController) ReplaceTransmit(oldTrackInfo, newTrackInfo *pb.ForwardTrack) error {
	// TODO: replace a track transition
	return nil
}
