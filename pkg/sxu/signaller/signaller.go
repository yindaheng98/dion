package signaller

import (
	"context"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pbrtc "github.com/pion/ion/proto/rtc"
	rtc "github.com/yindaheng98/dion/pkg/sxu/rtc"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
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
}

func (f SignallerFactory) NewDoor() (util.BlockedDoor, error) {
	return Signaller{
		cp:       f.cp,
		sfu:      f.sfu,
		Metadata: f.Metadata,
	}, nil
}

func NewSignallerFactory(cp ConnPool, sfu *ion_sfu.SFU) SignallerFactory {
	return SignallerFactory{
		cp:  cp,
		sfu: sfu,
	}
}

type Signaller struct {
	cp       ConnPool
	sfu      *ion_sfu.SFU
	Metadata metadata.MD

	r      *rtc.RTC
	cancel context.CancelFunc
}

func (s Signaller) BLock(param util.Param) error {
	track := param.Clone().(ForwardTrackParam).ForwardTrack
	conn := s.cp.GetConn(track.Src.Service, track.Src.Nid)
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

	peer := rtc.NewUpPeerLocal(ion_sfu.NewPeer(s.sfu))
	defer peer.Close()

	s.r = rtc.NewRTC(peer, signaller)
	return s.r.Run(track.RemoteSessionId, track.LocalSessionId)
}

func (s Signaller) Update(param util.Param) error {
	track := param.Clone().(ForwardTrackParam).ForwardTrack
	return s.r.Update(track.Tracks)
}

func (s Signaller) Remove() {
	if s.cancel != nil {
		s.cancel()
	}
}
