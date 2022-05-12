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
	pub   Publisher
	subs  map[string]Subscriber // TODO: Directly use WatchDog here
	subf  SubscriberFactory
}

func (b Bridge) Lock(init util.Param, OnBroken func(badGay error)) error {
	track := init.(ProceedTrackParam).ProceedTrack

	// Init Subscribers
	for _, sid := range track.SrcSessionIdList {
		if _, ok := b.subs[sid]; !ok {
			// make Subscriber
			subDoor, err := b.subf.NewDoor()
			if err != nil {
				return err
			}
			sub := subDoor.(Subscriber)
			// connect Subscriber to Publisher
			err = connect(sub, b.pub, OnBroken, init.(ProceedTrackParam).ProceedTrack)
			if err != nil {
				return err
			}
			b.subs[sid] = sub
		}
	}

	// start Publisher
	err := b.pub.Lock(SID(track.DstSessionId), OnBroken)
	if err != nil {
		return err
	}
	// start Subscribers
	for sid, sub := range b.subs {
		err := sub.Lock(SID(sid), OnBroken)
		if err != nil {
			return err
		}
	}
	return nil
}

func connect(src Subscriber, dst Publisher, OnBroken func(badGay error), track *pb.ProceedTrack) error {
	return nil
}

func (b Bridge) Repair(param util.Param, OnBroken func(error)) error {
	track := param.(ProceedTrackParam).ProceedTrack

	if track.DstSessionId != b.track.DstSessionId {
		return fmt.Errorf("destination SID not matched. current: %s, should be: %s", track.DstSessionId, b.track.DstSessionId)
	}
	panic("implement me")
}

func (b Bridge) Remove() {
	panic("implement me")
}
