package bridge

import (
	"fmt"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/protobuf/proto"
)

type BridgeFactory struct {
	pub PublisherFactory
	sub SubscriberFactory
}

func (b BridgeFactory) NewDoor() (util.Door, error) {
	panic("implement me")
}

type ProceedTrackParam struct {
	*pb.ProceedTrack
}

func (t ProceedTrackParam) Clone() util.Param {
	return ProceedTrackParam{ProceedTrack: proto.Clone(t.ProceedTrack).(*pb.ProceedTrack)}
}

type Bridge struct {
	track *pb.ProceedTrack
}

func (b Bridge) Repair(param util.Param) error {
	track := param.(ProceedTrackParam).ProceedTrack
	if track.DstSessionId != b.track.DstSessionId {
		return fmt.Errorf("destination SID not matched. current: %s, should be: %s", track.DstSessionId, b.track.DstSessionId)
	}
	panic("implement me")
}

func (b Bridge) Lock(init util.Param, OnBroken func(badGay error)) error {
	panic("implement me")
}

func (b Bridge) Remove() {
	panic("implement me")
}
