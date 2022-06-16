package util

import (
	"bufio"
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"io"
	"os/exec"
	"time"
)

func GetStdPipes(ffmpeg *exec.Cmd) (io.WriteCloser, io.ReadCloser, error) {
	ffmpegIn, err := ffmpeg.StdinPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StdinPipe(): %+v", err)
		return nil, nil, err
	}
	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StdoutPipe(): %+v", err)
		return nil, nil, err
	}
	ffmpegErr, err := ffmpeg.StderrPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StderrPipe(): %+v", err)
		return nil, nil, err
	}

	if err := ffmpeg.Start(); err != nil {
		log.Errorf("Cannot Start ffmpeg: %+v", err)
		return nil, nil, err
	}

	go func(ffmpegErr io.ReadCloser) {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}(ffmpegErr)
	return ffmpegIn, ffmpegOut, nil
}

func MakeIVFTrackFromStdout(ffmpegOut io.ReadCloser, codec webrtc.RTPCodecCapability) (webrtc.TrackLocal, error) {
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
		i := 0
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
			i += 1
			if i%30 == 0 {
				log.Debugf("%s %s Publish 30 RTP Packets to SFU TrackLocal", videoTrack.ID(), videoTrack.StreamID())
			}
		}
	}()
	return videoTrack, nil
}

func MakeSampleIVFTrack(ffmpegPath, Testsrc, Filter, Bandwidth string) (webrtc.TrackLocal, error) {
	// Create a video track
	videoopt := []string{
		"-f", "lavfi",
		"-i", "testsrc=" + Testsrc,
		"-vf", Filter,
		"-vcodec", "libvpx",
		"-b:v", Bandwidth,
		"-f", "ivf",
		"pipe:1",
	}
	ffmpegCmd := exec.Command(ffmpegPath, videoopt...) //nolint
	_, ffmpegOut, err := GetStdPipes(ffmpegCmd)
	if err != nil {
		return nil, err
	}
	return MakeIVFTrackFromStdout(ffmpegOut, webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8})
}
