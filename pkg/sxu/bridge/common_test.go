package bridge

import (
	"bufio"
	"context"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/spf13/viper"
	"github.com/yindaheng98/dion/util"
	"io"
	"os"
	"os/exec"
	"time"
)

type TestPublisherFactory struct {
	PublisherFactory
	ffmpegOut io.ReadCloser
}

func NewTestPublisherFactory(ffmpegOut io.ReadCloser, sfu *ion_sfu.SFU) TestPublisherFactory {
	return TestPublisherFactory{PublisherFactory: PublisherFactory{sfu: sfu}, ffmpegOut: ffmpegOut}
}

func (p TestPublisherFactory) NewDoor() (util.Door, error) {
	pub, err := p.PublisherFactory.NewDoor()
	if err != nil {
		log.Errorf("Cannot PublisherFactory.NewDoor: %+v", err)
		return nil, err
	}
	err = makeTrack(p.ffmpegOut, pub.(Publisher))
	if err != nil {
		log.Errorf("Cannot makeTrack: %+v", err)
		return nil, err
	}
	return pub, nil
}

func makeTrack(ffmpegOut io.ReadCloser, pub Publisher) error {
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

	pub.SetOnConnectionStateChange(func(err error) {
		fmt.Println("Peer Connection has gone to failed exiting")
	}, iceConnectedCtxCancel)

	return nil
}

const (
	portRangeLimit = 100
)

type global struct {
	Dc string `mapstructure:"dc"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

// Config defines parameters for the logger
type logConf struct {
	Level string `mapstructure:"level"`
}

// Config for sfu node
type Config struct {
	Global global   `mapstructure:"global"`
	Log    logConf  `mapstructure:"log"`
	Nats   natsConf `mapstructure:"nats"`
	ion_sfu.Config
}

func unmarshal(rawVal interface{}) error {
	if err := viper.Unmarshal(rawVal); err != nil {
		return err
	}
	return nil
}

func (c *Config) Load(file string) error {
	_, err := os.Stat(file)
	if err != nil {
		return err
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("config file %s read failed. %v\n", file, err)
		return err
	}

	err = unmarshal(c)
	if err != nil {
		return err
	}
	err = unmarshal(&c.Config)
	if err != nil {
		return err
	}
	if err != nil {
		log.Errorf("config file %s loaded failed. %v\n", file, err)
		return err
	}

	if len(c.WebRTC.ICEPortRange) > 2 {
		err = fmt.Errorf("config file %s loaded failed. range port must be [min,max]", file)
		log.Errorf("err=%v", err)
		return err
	}

	if len(c.WebRTC.ICEPortRange) != 0 && c.WebRTC.ICEPortRange[1]-c.WebRTC.ICEPortRange[0] < portRangeLimit {
		err = fmt.Errorf("config file %s loaded failed. range port must be [min, max] and max - min >= %d", file, portRangeLimit)
		log.Errorf("err=%v", err)
		return err
	}

	log.Infof("config %s load ok!", file)
	return nil
}

// readConf Read a Config
func readConf(confFile string) Config {
	conf := Config{}
	err := conf.Load(confFile)
	if err != nil {
		panic(err)
	}
	return conf
}

// makeVideo Make a video
func makeVideo(ffmpegPath, param string) io.ReadCloser {
	videoopt := []string{
		"-f", "lavfi",
		"-i", "testsrc=" + param,
		"-vf", "drawtext=text='%{localtime\\:%Y-%M-%d %H.%m.%S}' :fontsize=120",
		"-vcodec", "libvpx",
		"-b:v", "3M",
		"-f", "ivf",
		"pipe:1",
	}
	ffmpeg := exec.Command(ffmpegPath, videoopt...) //nolint
	ffmpegOut, _ := ffmpeg.StdoutPipe()
	ffmpegErr, _ := ffmpeg.StderrPipe()

	if err := ffmpeg.Start(); err != nil {
		panic(err)
	}

	go func() {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()
	return ffmpegOut
}
