package bridge

import (
	"bufio"
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	pb "github.com/yindaheng98/dion/proto"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Processor is a processor that can process webrtc.TrackRemote and output webrtc.TrackLocal
// MULTI-THREAD access!!! Should implemented in THREAD-SAFE!!!
type Processor interface {

	// Init set the AddTrack, RemoveTrack and OnBroken func and init the output track from Processor
	// Should be NON-BLOCK!
	// after you created a new track, please call AddTrack
	// before you close a track, please call RemoveTrack
	// when occurred error, please call OnBroken
	Init(
		AddTrack func(webrtc.TrackLocal) (*webrtc.RTPSender, error),
		RemoveTrack func(*webrtc.RTPSender) error,
		OnBroken func(badGay error),
	) error

	// AddInTrack add a input track to Processor
	// Will be called AFTER InitOutTrack!
	// read video from `remote` process it and write the result to the output track
	// r/w should stop when error occurred
	// Should be NON-BLOCK!
	AddInTrack(SID string, remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) error

	// UpdateProcedure update the procedure in the Processor
	UpdateProcedure(procedure *pb.ProceedTrack) error
}

type ProcessorFactory interface {
	NewProcessor() (Processor, error)
}

type SimpleFFmpegIVFProcessor struct {
	sync.Mutex
	ffmpegPath string
	Filter     string
	Bandwidth  string
	ffmpegIn   io.WriteCloser
	onBroken   func(badGay error)
}

func NewSimpleFFmpegIVFProcessor(ffmpegPath string) *SimpleFFmpegIVFProcessor {
	return &SimpleFFmpegIVFProcessor{
		ffmpegPath: ffmpegPath,
		Filter:     "drawbox=x=0:y=0:w=50:h=50:c=blue",
		Bandwidth:  "3M",
	}
}

func (t *SimpleFFmpegIVFProcessor) InitOutTrack(OnBroken func(badGay error)) (webrtc.TrackLocal, error) {
	t.Lock()
	defer t.Unlock()
	t.onBroken = OnBroken
	videoopt := []string{
		"-f", "ivf",
		"-i", "pipe:0",
		"-vf", t.Filter,
		"-vcodec", "libvpx",
		"-b:v", t.Bandwidth,
		"-f", "ivf",
		"pipe:1",
	}
	ffmpeg := exec.Command(t.ffmpegPath, videoopt...) //nolint
	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StdoutPipe(): %+v", err)
		return nil, err
	}
	t.ffmpegIn, err = ffmpeg.StdinPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StdinPipe(): %+v", err)
		return nil, err
	}
	ffmpegErr, err := ffmpeg.StderrPipe()
	if err != nil {
		log.Errorf("Cannot get ffmpeg.StderrPipe(): %+v", err)
		return nil, err
	}
	go func(ffmpegErr io.ReadCloser) {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}(ffmpegErr)

	if err := ffmpeg.Start(); err != nil {
		log.Errorf("Cannot Start ffmpeg: %+v", err)
		return nil, err
	}

	videoTrack, videoTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "SimpleFFmpegProcessor", "SimpleFFmpegProcessor")
	if videoTrackErr != nil {
		return nil, videoTrackErr
	}

	go func() {
		ivf, header, ivfErr := ivfreader.NewWith(ffmpegOut)
		if ivfErr != nil {
			log.Errorf("ivfreader create error: %+v", ivfErr)
			OnBroken(ivfErr)
			return
		}

		ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000))
		for ; true; <-ticker.C {
			frame, _, ivfErr := ivf.ParseNextFrame()
			if ivfErr == io.EOF {
				log.Errorf("All video frames parsed and sent")
				OnBroken(ivfErr)
				return
			}

			if ivfErr != nil {
				log.Errorf("Video frames parse error: %+v", ivfErr)
				OnBroken(ivfErr)
				return
			}

			if ivfErr = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); ivfErr != nil {
				log.Errorf("Video frames write error: %+v", ivfErr)
				OnBroken(ivfErr)
				return
			}
		}
	}()
	return videoTrack, nil
}

func (t *SimpleFFmpegIVFProcessor) AddInTrack(_ string, remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) error {
	t.Lock()
	defer t.Unlock()
	if t.ffmpegIn == nil {
		log.Warnf("SimpleFFmpegIVFProcessor.AddInTrack should not be called twice!!!")
		return nil
	}
	ffmpegIn := t.ffmpegIn
	t.ffmpegIn = nil
	ivfWriter, err := ivfwriter.NewWith(ffmpegIn)
	if err != nil {
		log.Errorf("Cannot create ivfwriter: %+v", err)
		return err
	}
	fmt.Println("Track from SFU added")
	go func(remote *webrtc.TrackRemote, ivfWriter *ivfwriter.IVFWriter) {
		for {
			// Read RTP packets being sent to Pion
			rtp, _, readErr := remote.ReadRTP()
			fmt.Println("RTP Packat read from SFU")
			if readErr != nil {
				log.Errorf("RTP Packat read error: %+v", readErr)
				t.onBroken(readErr)
				return
			}

			if ivfWriterErr := ivfWriter.WriteRTP(rtp); ivfWriterErr != nil {
				log.Errorf("RTP Packat write error: %+v", ivfWriterErr)
				t.onBroken(ivfWriterErr)
				return
			}
		}
	}(remote, ivfWriter)

	return nil
}

func (t *SimpleFFmpegIVFProcessor) UpdateProcedure(procedure *pb.ProceedTrack) error {
	fmt.Printf("SimpleProcessor Updating: %+v\n", procedure)
	return nil
}
