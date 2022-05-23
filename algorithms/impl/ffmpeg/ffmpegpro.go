package ffmpeg

import (
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/yindaheng98/dion/algorithms"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"io"
	"os/exec"
)

type SimpleFFmpegIVFProcessorFactory struct {
	ffmpegPath string
	Filter     string
	Bandwidth  string
}

func NewSimpleFFmpegIVFProcessorFactory(ffmpegPath string) *SimpleFFmpegIVFProcessorFactory {
	return &SimpleFFmpegIVFProcessorFactory{
		ffmpegPath: ffmpegPath,
		Filter:     "drawbox=x=0:y=0:w=50:h=50:c=blue",
		Bandwidth:  "3M",
	}
}

func (s SimpleFFmpegIVFProcessorFactory) NewProcessor() (algorithms.Processor, error) {
	return &SimpleFFmpegIVFProcessor{
		ffmpegPath: s.ffmpegPath,
		Filter:     "drawbox=x=0:y=0:w=50:h=50:c=blue",
		Bandwidth:  "3M",
	}, nil
}

type SimpleFFmpegIVFProcessor struct {
	ffmpegPath string
	Filter     string
	Bandwidth  string

	addTrack    func(webrtc.TrackLocal) (*webrtc.RTPSender, error)
	removeTrack func(*webrtc.RTPSender) error
	onBroken    func(badGay error)
}

func (t *SimpleFFmpegIVFProcessor) Init(AddTrack func(webrtc.TrackLocal) (*webrtc.RTPSender, error), RemoveTrack func(*webrtc.RTPSender) error, OnBroken func(badGay error)) error {
	t.addTrack = AddTrack
	t.removeTrack = RemoveTrack
	t.onBroken = OnBroken
	return nil
}

func WriteIVFRemoteToStdin(remote *webrtc.TrackRemote, stdin io.WriteCloser, ffmpeg *exec.Cmd, OnBroken func(error)) error {
	ivfWriter, err := ivfwriter.NewWith(stdin)
	if err != nil {
		log.Errorf("Cannot create ivfwriter: %+v", err)
		return err
	}
	go func(remote *webrtc.TrackRemote, ivfWriter *ivfwriter.IVFWriter, OnBroken func(error)) {
		for {
			// Read RTP packets being sent to Pion
			rtp, _, readErr := remote.ReadRTP()
			fmt.Println(remote.ID(), remote.StreamID(), "Read RTP Packet from SFU TrackRemote")
			if readErr != nil {
				if err := ffmpeg.Process.Kill(); err != nil {
					log.Errorf("Cannot kill: %+v", err)
				}
				OnBroken(readErr)
				return
			}

			if ivfWriterErr := ivfWriter.WriteRTP(rtp); ivfWriterErr != nil {
				log.Errorf("RTP Packet write error: %+v", ivfWriterErr)
				if err := ffmpeg.Process.Kill(); err != nil {
					log.Errorf("Cannot kill: %+v", err)
				}
				OnBroken(ivfWriterErr)
				return
			}
		}
	}(remote, ivfWriter, OnBroken)
	return nil
}

func (t *SimpleFFmpegIVFProcessor) AddInTrack(_ string, remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) error {
	// Create a video track
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
	ffmpegIn, ffmpegOut, err := util.GetStdPipes(ffmpeg)
	if err != nil {
		return err
	}
	senderCh := make(chan *webrtc.RTPSender, 1)
	go func(senderCh chan<- *webrtc.RTPSender) {
		track, err := util.MakeIVFTrackFromStdout(ffmpegOut, webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8})
		if err != nil {
			close(senderCh)
			return
		}
		sender, err := t.addTrack(track)
		if err != nil {
			close(senderCh)
			return
		}
		senderCh <- sender
	}(senderCh)
	return WriteIVFRemoteToStdin(remote, ffmpegIn, ffmpeg, func(err error) {
		log.Errorf("Should remove track: %+v", err)
		sender, ok := <-senderCh
		if !ok {
			return
		}
		if err := t.removeTrack(sender); err != nil {
			log.Errorf("Cannot remove track: %+v", err)
		}
	})
}

func (t *SimpleFFmpegIVFProcessor) UpdateProcedure(procedure *pb.ProceedTrack) error {
	fmt.Printf("SimpleProcessor Updating: %+v\n", procedure)
	return nil
}
