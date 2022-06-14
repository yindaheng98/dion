package main

import (
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/yindaheng98/dion/util"
	"io"
	"time"
)

func PlayIt(ffmpegOut io.ReadCloser, ffplayIn io.WriteCloser) io.Reader {
	return io.TeeReader(ffmpegOut, ffplayIn)
}

func SendIt(ffmpegOut io.Reader, codec webrtc.RTPCodecCapability) (webrtc.TrackLocal, error) {
	ivf, header, err := ivfreader.NewWith(ffmpegOut)
	if err != nil {
		log.Errorf("ivfreader create error: %+v", err)
		return nil, err
	}

	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		codec,
		"SampleIVFTrack-"+util.RandomString(8),
		"SampleIVFTrack-"+util.RandomString(8),
	)
	if err != nil {
		log.Errorf("Cannot webrtc.NewTrackLocalStaticSample: %+v", err)
		return nil, err
	}

	go func() {
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
				log.Errorf("All video frames parsed and sent")
				return
			}
			if ivfErr != nil {
				log.Errorf("Cannot ParseNextFrame: %+v", ivfErr)
				return
			}
			if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil { // 这个track被remove了也不会报错，这就是停不下来的原因
				log.Errorf("Cannot WriteSample: %+v", ivfErr)
				return
			}
		}
	}()
	return videoTrack, nil
}
