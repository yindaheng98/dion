package router

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/signaller"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
)

type forwarding struct {
	util.WatchDog
	util.ForwardTrackItem
}

type ForwardRouter struct {
	factory     signaller.SignallerFactory
	forwardings map[string]forwarding // map<NID, map<SID, forwarding>>
}

func NewForwardRouter(sfu *ion_sfu.SFU, cp signaller.ConnPool, with ...func(signaller.SignallerFactory)) ForwardRouter {
	return ForwardRouter{
		factory:     signaller.NewSignallerFactory(cp, sfu, with...),
		forwardings: map[string]forwarding{},
	}
}

func (f ForwardRouter) StartForwardTrack(trackInfo *pb.ForwardTrack) {
	item := util.ForwardTrackItem{Track: trackInfo}
	proc, ok := f.forwardings[item.Key()]
	if ok { // peer exist?
		f.ReplaceForwardTrack(proc.Track, trackInfo) // if exist, just update
		return
	}

	proc = forwarding{
		WatchDog:         util.NewWatchDogWithBlockedDoor(f.factory),
		ForwardTrackItem: item.Clone().(util.ForwardTrackItem),
	}
	proc.Watch(signaller.ForwardTrackParam{ForwardTrack: item.Clone().(util.ForwardTrackItem).Track})
	f.forwardings[item.Key()] = proc
}

func (f ForwardRouter) StopForwardTrack(trackInfo *pb.ForwardTrack) {
	item := util.ForwardTrackItem{Track: trackInfo}
	if proc, ok := f.forwardings[item.Key()]; ok { // peer exist?
		proc.Leave() // if exist, just stop
		delete(f.forwardings, item.Key())
	}
}

func (f ForwardRouter) ReplaceForwardTrack(oldTrackInfo *pb.ForwardTrack, newTrackInfo *pb.ForwardTrack) {
	olditem := util.ForwardTrackItem{Track: oldTrackInfo}
	newitem := util.ForwardTrackItem{Track: newTrackInfo}
	if olditem.Key() != newitem.Key() {
		log.Warnf("Cannot ReplaceForwardTrack when util.ForwardTrackItem.Key() is not same")
		return
	}
	proc, ok := f.forwardings[olditem.Key()]
	if !ok { // peer not exist?
		return // just return
	}
	proc.Update(signaller.ForwardTrackParam{ForwardTrack: newitem.Clone().(util.ForwardTrackItem).Track})
	proc.Track = newitem.Clone().(util.ForwardTrackItem).Track
}
