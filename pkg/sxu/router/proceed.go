package router

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
)

type proceeding struct {
	util.WatchDog
	util.ProceedTrackItem
}

// ProceedRouter controls the track proceed in SFU
type ProceedRouter struct {
	sfu         *ion_sfu.SFU
	factory     ProcessorFactory
	proceedings map[string]proceeding
}

func NewProceedRouter(sfu *ion_sfu.SFU, factory ProcessorFactory) ProceedRouter {
	return ProceedRouter{
		sfu:         sfu,
		factory:     factory,
		proceedings: map[string]proceeding{},
	}
}

func (p ProceedRouter) StartProceedTrack(trackInfo *pb.ProceedTrack) {
	item := util.ProceedTrackItem{Track: trackInfo}
	proc, ok := p.proceedings[item.Key()]
	if ok { // peer exist?
		p.ReplaceProceedTrack(proc.Track, trackInfo) // if exist, just update
		return
	}
	pro := p.factory.NewProcessor(item.Clone().(util.ProceedTrackItem).Track)
	proc = proceeding{
		WatchDog:         util.NewUnblockedWatchDog(bridge.NewBridgeFactory(p.sfu, pro)),
		ProceedTrackItem: item.Clone().(util.ProceedTrackItem),
	}
	proc.Watch(bridge.ProceedTrackParam{ProceedTrack: item.Clone().(util.ProceedTrackItem).Track})
	p.proceedings[item.Key()] = proc
}

func (p ProceedRouter) StopProceedTrack(trackInfo *pb.ProceedTrack) {
	item := util.ProceedTrackItem{Track: trackInfo}
	if proc, ok := p.proceedings[item.Key()]; ok { // peer exist?
		proc.Leave() // if exist, just stop
		delete(p.proceedings, item.Key())
	}
}

func (p ProceedRouter) ReplaceProceedTrack(oldTrackInfo *pb.ProceedTrack, newTrackInfo *pb.ProceedTrack) {
	olditem := util.ProceedTrackItem{Track: oldTrackInfo}
	newitem := util.ProceedTrackItem{Track: newTrackInfo}
	if olditem.Key() != newitem.Key() {
		log.Warnf("Cannot ReplaceProceedTrack when util.ProceedTrackItem.Key() is not same")
		return
	}
	proc, ok := p.proceedings[olditem.Key()]
	if !ok { // peer not exist?
		return // just return
	}
	proc.Update(bridge.ProceedTrackParam{ProceedTrack: newitem.Clone().(util.ProceedTrackItem).Track})
	proc.Track = newitem.Clone().(util.ProceedTrackItem).Track
}
