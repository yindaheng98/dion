package signaller

import (
	"context"
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pbrtc "github.com/pion/ion/proto/rtc"
	rtc "github.com/yindaheng98/dion/pkg/sxu/rtc"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"sync"
)

type ForwardTrackParam struct {
	*pb.ForwardTrack
}

func (t ForwardTrackParam) Clone() util.Param {
	return ForwardTrackParam{ForwardTrack: proto.Clone(t.ForwardTrack).(*pb.ForwardTrack)}
}

type SignallerFactory struct {
	cp       ConnPool
	sfu      *ion_sfu.SFU
	Metadata metadata.MD
	IRFBF    PubIRFBuilderFactory
}

func (f SignallerFactory) NewDoor() (util.BlockedDoor[ForwardTrackParam], error) {
	return &Signaller{
		cp:       f.cp,
		sfu:      f.sfu,
		Metadata: f.Metadata,
		irfbf:    f.IRFBF,
	}, nil
}

func NewSignallerFactory(cp ConnPool, sfu *ion_sfu.SFU) SignallerFactory {
	sf := SignallerFactory{
		cp:  cp,
		sfu: sfu,
	}
	return sf
}

type Signaller struct {
	cp    ConnPool
	irfbf PubIRFBuilderFactory

	sfu      *ion_sfu.SFU
	Metadata metadata.MD

	r      *rtc.RTC
	cancel context.CancelFunc

	track   *pb.ForwardTrack
	trackMu sync.Mutex
}

func (s *Signaller) BLock(param ForwardTrackParam) error {
	track := param.Clone().(ForwardTrackParam).ForwardTrack
	s.track = track
	conn, err := s.cp.GetConn(track.Src.Service, track.Src.Nid)
	if err != nil {
		return err
	}
	client := pbrtc.NewRTCClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.cancel = cancel

	// Initialize GRPC signaller
	ctx = metadata.NewOutgoingContext(ctx, s.Metadata)
	signaller, err := client.Signal(ctx)
	if err != nil {
		return err
	}
	defer signaller.CloseSend()

	var irfb ion_sfu.InterceptorRegistryFactoryBuilder
	if s.irfbf != nil {
		irfb = s.irfbf.NewBuilder(track.Src)
	}

	peer := rtc.NewUpPeerLocal(ion_sfu.NewPeer(s.sfu, ion_sfu.WithPubInterceptorRegistryFactoryBuilder(irfb)))
	defer peer.Close()

	s.r = rtc.NewRTC(peer, signaller)
	return s.r.Run(track.RemoteSessionId, track.LocalSessionId)
}

func TrackSame(track1, track2 *pb.ForwardTrack) bool {
	if track1 == nil || track2 == nil {
		return true
	}
	if track1.Src.Nid != track2.Src.Nid {
		return false
	}
	if track1.RemoteSessionId != track2.RemoteSessionId {
		return false
	}
	if track1.LocalSessionId != track2.LocalSessionId {
		return false
	}
	return true
}

func (s *Signaller) Update(param ForwardTrackParam) error {
	oldTrack := s.track
	track := param.Clone().(ForwardTrackParam).ForwardTrack
	// should pull from another remote session? or should push to another local session?
	if !TrackSame(oldTrack, track) {
		log.Warnf("Track is not same, cannot update, should restart")
		s.Remove() // update cannot handle it, should restart
		return nil
	}
	if s.r == nil {
		log.Warnf("Cannot update: peer not start")
		return fmt.Errorf("peer not start")
	}
	return s.r.Update(track.Tracks)
}

func (s *Signaller) Remove() {
	if s.cancel != nil {
		s.cancel()
	}
}
