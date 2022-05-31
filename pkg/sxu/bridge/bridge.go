package bridge

import (
	"fmt"
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/algorithms"
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
	algorithms.ProcessorFactory

	sub SubscriberFactory // to create entrance
}

func NewBridgeFactory(sfu *ion_sfu.SFU, fact algorithms.ProcessorFactory) BridgeFactory {
	return BridgeFactory{
		PublisherFactory: NewPublisherFactory(sfu),
		ProcessorFactory: fact,
		sub:              NewSubscriberFactory(sfu),
	}
}

func (b BridgeFactory) NewDoor() (util.UnblockedDoor[ProceedTrackParam], error) {
	pubDoor, err := b.PublisherFactory.NewDoor()
	if err != nil {
		return nil, err
	}
	pub := pubDoor.(Publisher)
	pro, err := b.ProcessorFactory.NewProcessor()
	if err != nil {
		return nil, err
	}
	return Bridge{
		EntranceFactory: EntranceFactory{
			SubscriberFactory: b.sub,
			road:              pro,
		},
		Processor: pro,
		exit:      pub,
		track:     nil,
		entrances: map[string]util.WatchDog[SID]{},
	}, nil
}

type Bridge struct {
	EntranceFactory // Bridge should have the ability to generate Entrance
	// But Bridge only have 1 Publisher in EntranceFactory, if this Publisher broken, the the bridge broken
	algorithms.Processor
	exit Publisher

	track     *pb.ProceedTrack
	entrances map[string]util.WatchDog[SID]
}

func (b Bridge) Update(param ProceedTrackParam) error {
	track := param.Clone().(ProceedTrackParam).ProceedTrack           // Clone it
	if b.track != nil && b.track.DstSessionId != track.DstSessionId { // check if it is mine
		log.Errorf("DstSessionId not match! ")
		return fmt.Errorf("DstSessionId not match! ")
	}
	b.track = param.Clone().(ProceedTrackParam).ProceedTrack // Store it

	// Store it in Processor
	err := b.Processor.UpdateProcedure(track)
	if err != nil {
		return err
	}

	// Remove the deprecated sessions and Create missing sessions

	// Create missing sessions
	for _, sid := range track.SrcSessionIdList {
		if _, ok := b.entrances[sid]; !ok { // missing?
			// make Entrance watchdog
			entrance := util.NewWatchDogWithUnblockedDoor[SID](b.EntranceFactory)
			entrance.Watch(SID(sid)) // start it
			b.entrances[sid] = entrance
		}
	}

	// Remove the deprecated sessions
	sidSet := map[string]bool{}
	for _, sid := range track.SrcSessionIdList {
		sidSet[sid] = true // Record the expected sessions
	}
	for sid, entrance := range b.entrances { // Fine deprecated sessions
		if _, ok := sidSet[sid]; !ok { // if it is deprecated session
			entrance.Leave()         // stop it
			delete(b.entrances, sid) // remove it
		}
	}

	return nil
}

func (b Bridge) Lock(init ProceedTrackParam, OnBroken func(badGay error)) error {
	// start Publisher
	track := init.ProceedTrack
	err := b.exit.Lock(SID(track.DstSessionId), OnBroken)
	if err != nil {
		return err
	}

	// Init the Processor
	err = b.Processor.Init(
		func(videoTrack webrtc.TrackLocal) (*webrtc.RTPSender, error) {
			rtpSender, err := b.exit.AddTrack(videoTrack)
			if err != nil {
				return nil, err
			}

			// Read incoming RTCP packets
			// Before these packets are returned they are processed by interceptors. For things
			// like NACK this needs to be called.
			go func(rtpSender *webrtc.RTPSender) {
				rtcpBuf := make([]byte, 1500)
				for {
					if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
						return
					}
				}
			}(rtpSender)

			return rtpSender, nil
		},
		func(sender *webrtc.RTPSender) error {
			err := b.exit.RemoveTrack(sender)
			if err != nil {
				log.Errorf("Cannot remove track: %+v", err)
				return err
			}
			return nil
		},
		OnBroken)

	if err != nil {
		return err
	}

	return b.Update(init)
}

func (b Bridge) Repair(param ProceedTrackParam) error {
	return fmt.Errorf("Bridge cannot be repaired ")
}

func (b Bridge) Remove() {
	// start Subscribers
	for sid, entrance := range b.entrances {
		entrance.Leave()
		delete(b.entrances, sid)
	}
	b.exit.Remove()
}
