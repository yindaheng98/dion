package sxu

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/islb"
	"github.com/yindaheng98/dion/pkg/sxu/signaller"
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
)

type forwarding struct {
	util.WatchDog[signaller.ForwardTrackParam]
	util.ForwardTrackItem
}

type ForwardRouter struct {
	factory     signaller.SignallerFactory
	forwardings map[string]forwarding // map<NID, map<SID, forwarding>>
}

type ForwardRouterOption func(ForwardRouter)

func WithPubIRFBuilderFactory(irfbf signaller.PubIRFBuilderFactory) ForwardRouterOption {
	return func(r ForwardRouter) {
		r.factory.IRFBF = irfbf
	}
}

func WithTrackForwarder(with ...ForwardRouterOption) WithOption {
	return func(box *syncer.ToolBox, node *islb.Node, sfu *ion_sfu.SFU) {
		TrackForwarder := NewForwardRouter(sfu, NewNRPCConnPool(node))
		for _, w := range with {
			w(TrackForwarder)
		}
		box.TrackForwarder = TrackForwarder
	}
}

func NewForwardRouter(sfu *ion_sfu.SFU, cp signaller.ConnPool, with ...func(ForwardRouter)) ForwardRouter {
	r := ForwardRouter{
		factory:     signaller.NewSignallerFactory(cp, sfu),
		forwardings: map[string]forwarding{},
	}
	for _, w := range with {
		w(r)
	}
	return r
}

func (f ForwardRouter) StartForwardTrack(trackInfo *pb.ForwardTrack) {
	item := util.ForwardTrackItem{Track: trackInfo}
	proc, ok := f.forwardings[item.Key()]
	if ok { // peer exist?
		f.ReplaceForwardTrack(proc.Track, trackInfo) // if exist, just update
		return
	}

	proc = forwarding{
		WatchDog:         util.NewWatchDogWithBlockedDoor[signaller.ForwardTrackParam](f.factory),
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
