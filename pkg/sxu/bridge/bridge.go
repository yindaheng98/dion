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
	track := param.Clone().(ProceedTrackParam).ProceedTrack           // Clone it
	if b.track != nil && b.track.DstSessionId != track.DstSessionId { // check if it is mine
		return fmt.Errorf("DstSessionId not match! ")
	}

	// Update it
	sidSet := map[string]bool{} // Record the expected sessions
	// Init Subscribers
	for _, sid := range track.SrcSessionIdList {
		sidSet[sid] = true // Record the expected sessions
		if _, ok := b.entrances[sid]; !ok {
			// make Entrance watchdog
			b.entrances[sid] = util.NewWatchDog(b.fact)
		}
	}
	b.Processor.UpdateProcedure(track)

	// start Subscribers
	for sid, entrance := range b.entrances {
		if _, ok := sidSet[sid]; !ok { // if it is unexpected session
			entrance.Leave() // stop it
			delete(b.entrances, sid)
		} else {
			entrance.Watch(SID(sid))
		}
	}

	b.track = param.Clone().(ProceedTrackParam).ProceedTrack // Store it
	return nil
}

func (b Bridge) Lock(init util.Param, OnBroken func(badGay error)) error {
	// start Publisher
	track := init.(ProceedTrackParam).ProceedTrack
	err := b.fact.exit.Lock(SID(track.DstSessionId), OnBroken)
	if err != nil {
		return err
	}
	return b.Update(init, OnBroken)
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
