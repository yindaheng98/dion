package rtc

import (
	"encoding/json"
	"errors"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓

func (r *RTC) onSingalHandleOnce() {
	// onSingalHandle is wrapped in a once and only started after another public
	// method is called to ensure the user has the opportunity to register handlers
	r.handleOnce.Do(func() {
		err := r.onSingalHandle()
		if r.OnError != nil {
			r.OnError(err)
		}
	})
}

func (r *RTC) onSingalHandle() error {
	for {
		//only one goroutine for recving from stream, no need to lock
		stream, err := r.signaller.Recv()
		if err != nil {
			if err == io.EOF {
				log.Infof("[%v] WebRTC Transport Closed", r.uid)
				if err := r.signaller.CloseSend(); err != nil {
					log.Errorf("[%v] error sending close: %s", r.uid, err)
				}
				return err
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				if err := r.signaller.CloseSend(); err != nil {
					log.Errorf("[%v] error sending close: %s", r.uid, err)
				}
				return err
			}

			log.Errorf("[%v] Error receiving RTC response: %v", r.uid, err)
			/*
				if r.OnError != nil {
					r.OnError(err)
				}
			*/
			return err
		}

		switch payload := stream.Payload.(type) {
		case *rtc.Reply_Join:
			success := payload.Join.Success
			err := errors.New(payload.Join.Error.String())

			if !success {
				log.Errorf("[%v] [join] failed error: %v", r.uid, err)
				return err
			}
			log.Infof("[%v] [join] success", r.uid)
			/*
				log.Infof("payload.Reply.Description=%v", payload.Join.Description)
				sdp := webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  payload.Join.Description.Sdp,
				}

				if err = r.setRemoteSDP(sdp); err != nil {
					log.Errorf("[%v] [join] error %s", r.uid, err)
					return err
				}
			*/
		case *rtc.Reply_Description:
			var sdpType webrtc.SDPType
			if payload.Description.Type == "offer" {
				sdpType = webrtc.SDPTypeOffer
			} else {
				sdpType = webrtc.SDPTypeAnswer
			}
			sdp := webrtc.SessionDescription{
				SDP:  payload.Description.Sdp,
				Type: sdpType,
			}
			if sdp.Type == webrtc.SDPTypeOffer {
				log.Infof("[%v] [description] got offer call s.OnNegotiate sdp=%+v", r.uid, sdp)
				err := r.negotiate(sdp)
				if err != nil {
					// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑
					log.Errorf("error when negotiate: %v", err)
					return err
				}
			} else if sdp.Type == webrtc.SDPTypeAnswer {
				log.Infof("[%v] [description] got answer call sdp=%+v, but i do not have a publisher", r.uid, sdp)
			}
			// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓
		case *rtc.Reply_Trickle:
			var candidate webrtc.ICECandidateInit
			_ = json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			log.Infof("[%v] [trickle] type=%v candidate=%v", r.uid, payload.Trickle.Target, candidate)
			// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑
			err := r.trickle(candidate, Target(payload.Trickle.Target))
			if err != nil {
				log.Errorf("error when trickle: %v", err)
				return err
			}
			// ↓↓↓↓↓ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↓↓↓↓↓
		case *rtc.Reply_TrackEvent:
			/*
				if r.OnTrackEvent == nil {
					log.Errorf("s.OnTrackEvent == nil")
					continue
				}
				var TrackInfos []*TrackInfo
				for _, v := range payload.TrackEvent.Tracks {
					TrackInfos = append(TrackInfos, &TrackInfo{
						Id:        v.Id,
						Kind:      v.Kind,
						Muted:     v.Muted,
						Type:      MediaType(v.Type),
						StreamId:  v.StreamId,
						Label:     v.Label,
						Width:     v.Width,
						Height:    v.Height,
						FrameRate: v.FrameRate,
						Layer:     v.Layer,
					})
				}
				trackEvent := TrackEvent{
					State:  TrackEvent_State(payload.TrackEvent.State),
					Uid:    payload.TrackEvent.Uid,
					Tracks: TrackInfos,
				}

				log.Infof("s.OnTrackEvent trackEvent=%+v", trackEvent)
				r.OnTrackEvent(trackEvent)
			*/
		case *rtc.Reply_Subscription:
			if !payload.Subscription.Success {
				log.Errorf("suscription error: %v", payload.Subscription.Error)
				return errors.New(payload.Subscription.Error.String())
			}
		case *rtc.Reply_Error:
			log.Errorf("Request error: %v", payload.Error)
			return errors.New(payload.Error.String())
		default:
			log.Errorf("Unknown RTC type!!!!%v", payload)
		}
	}
}

// ↑↑↑↑↑ Copy from: https://github.com/pion/ion-sdk-go/blob/12e32a5871b905bf2bdf58bc45c2fdd2741c4f81/rtc.go ↑↑↑↑↑
