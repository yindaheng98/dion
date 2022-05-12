package main

import (
	"context"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	"github.com/yindaheng98/dion/util"
	"io"
	"os"
	"time"
)

type PublisherFactory struct {
	sfu       *ion_sfu.SFU
	ffmpegOut io.ReadCloser
}

func NewPublisherFactory(ffmpegOut io.ReadCloser, sfu *ion_sfu.SFU) PublisherFactory {
	return PublisherFactory{ffmpegOut: ffmpegOut, sfu: sfu}
}

func (p PublisherFactory) NewDoor() (util.Door, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Errorf("Cannot NewPeerConnection: %+v", err)
		return nil, err
	}
	pub := bridge.Publisher{BridgePeer: bridge.NewBridgePeer(ion_sfu.NewPeer(p.sfu), pc)}
	err = makeTrack(p.ffmpegOut, pub)
	if err != nil {
		log.Errorf("Cannot makeTrack: %+v", err)
		return nil, err
	}
	return pub, nil
}

func makeTrack(ffmpegOut io.ReadCloser, pub bridge.Publisher) error {
	// Create a video track
	videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if videoTrackErr != nil {
		return videoTrackErr
	}

	rtpSender, videoTrackErr := pub.AddTrack(videoTrack)
	if videoTrackErr != nil {
		return videoTrackErr
	}

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(context.Background())

	go func() {
		ivf, header, ivfErr := ivfreader.NewWith(ffmpegOut)
		if ivfErr != nil {
			panic(ivfErr)
		}

		// Wait for connection established
		<-iceConnectedCtx.Done()

		// Send our video file frame at a time. Pace our sending so we send it at the same speed it should be played back as.
		// This isn't required since the video is timestamped, but we will such much higher loss if we send all at once.
		//
		// It is important to use a time.Ticker instead of time.Sleep because
		// * avoids accumulating skew, just calling time.Sleep didn't compensate for the time spent parsing the data
		// * works around latency issues with Sleep (see https://github.com/golang/go/issues/44343)
		ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000))
		for ; true; <-ticker.C {
			frame, _, ivfErr := ivf.ParseNextFrame()
			if ivfErr == io.EOF {
				fmt.Printf("All video frames parsed and sent")
				os.Exit(0)
			}

			if ivfErr != nil {
				panic(ivfErr)
			}

			if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
				panic(ivfErr)
			}
		}
	}()

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	pub.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateConnected {
			iceConnectedCtxCancel()
		}
	})

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	pub.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}
	})
	return nil
}
