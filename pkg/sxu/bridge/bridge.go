package bridge

import (
	"fmt"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/protobuf/proto"
)

type ProceedTrackParam struct {
	*pb.ProceedTrack
}

func (t ProceedTrackParam) Clone() util.Param {
	return ProceedTrackParam{ProceedTrack: proto.Clone(t.ProceedTrack).(*pb.ProceedTrack)}
}

type BridgeFactory struct {
	PublisherFactory
	sub SubscriberFactory
	pro Processor
}

func NewBridgeFactory(sfu *ion_sfu.SFU, pro Processor) BridgeFactory {
	return BridgeFactory{
		PublisherFactory: NewPublisherFactory(sfu),
		sub:              NewSubscriberFactory(sfu),
		pro:              pro,
	}
}

func (b BridgeFactory) NewDoor() (util.Door, error) {
	pubDoor, err := b.PublisherFactory.NewDoor()
	if err != nil {
		return nil, err
	}
	pub := pubDoor.(Publisher)
	return Bridge{
		fact: EntranceFactory{
			SubscriberFactory: b.sub,
			exit:              pub,
			road:              b.pro,
		},
		Processor: b.pro,
		track:     nil,
		entrances: map[string]*util.WatchDog{},
	}, nil
}

type Bridge struct {
	fact EntranceFactory // Bridge should have the ability to generate Entrance
	// But Bridge only have 1 Publisher in EntranceFactory, if this Publisher broken, the the bridge broken
	Processor

	track     *pb.ProceedTrack
	entrances map[string]*util.WatchDog
}

func (b Bridge) Update(param util.Param, OnBroken func(badGay error)) error {
	track := param.Clone().(ProceedTrackParam).ProceedTrack
	if b.track != nil && b.track.DstSessionId != track.DstSessionId {
		return fmt.Errorf("DstSessionId not match! ")
	}
	b.Processor.UpdateProcedure(track)
	b.track = param.Clone().(ProceedTrackParam).ProceedTrack
	return nil
}

func (b Bridge) Lock(init util.Param, OnBroken func(badGay error)) error {
	track := init.Clone().(ProceedTrackParam).ProceedTrack

	// Init Subscribers
	for _, sid := range track.SrcSessionIdList {
		if _, ok := b.entrances[sid]; !ok {
			// make Entrance watchdog
			b.entrances[sid] = util.NewWatchDog(b.fact)
		}
	}

	// start Publisher
	err := b.fact.exit.Lock(SID(track.DstSessionId), OnBroken)
	if err != nil {
		return err
	}
	b.Processor.UpdateProcedure(track)
	b.track = init.Clone().(ProceedTrackParam).ProceedTrack
	// start Subscribers
	for sid, entrance := range b.entrances {
		entrance.Watch(SID(sid))
	}
	return nil
}

func (b Bridge) Repair(param util.Param, OnBroken func(error)) error {
	return fmt.Errorf("Bridge cannot be repaired ")
}

func (b Bridge) Remove() {
	// start Subscribers
	for _, entrance := range b.entrances {
		entrance.Leave()
	}
	b.fact.exit.Remove()
}
