package bridge

import (
	"fmt"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/algorithms"
	"github.com/yindaheng98/dion/util"
)

// EntranceFactory create Entrance for a bridge
// All the Entrance(or in other words, Subscriber) should put their track into same Processor
type EntranceFactory struct {
	SubscriberFactory
	road algorithms.Processor
}

func (e EntranceFactory) NewDoor() (util.UnblockedDoor[SID], error) {
	sub, err := e.SubscriberFactory.NewDoor()
	if err != nil {
		return nil, err
	}
	return Entrance{
		Subscriber: sub.(Subscriber),
		road:       e.road,
	}, nil
}

// Entrance of a Bridge
type Entrance struct {
	Subscriber // Subscriber is its entrance, Entrance is also a Subscriber
	road       algorithms.Processor
}

func (e Entrance) Lock(init SID, OnBroken func(badGay error)) error {
	sid := init

	e.Subscriber.OnTrack(func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		err := e.road.AddInTrack(string(sid), remote, receiver)
		if err != nil {
			OnBroken(err)
		}
	})

	return e.Subscriber.Lock(sid, OnBroken)
}

func (e Entrance) Repair(param SID) error {
	return fmt.Errorf("Entrance cannot be repaired ")
}

func (e Entrance) Update(param SID) error {
	return fmt.Errorf("Entrance cannot be updated ")
}

func (e Entrance) Remove() {
	e.Subscriber.Remove()
}
