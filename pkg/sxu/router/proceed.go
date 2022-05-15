package router

import (
	log "github.com/pion/ion-log"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/protobuf/proto"
)

type proceeding struct {
	*util.WatchDog
	info *pb.ProceedTrack
}

// ProceedRouter controls the track proceed in SFU
type ProceedRouter struct {
	sfu         *ion_sfu.SFU
	factory     ProcessorFactory
	proceedings map[string]proceeding
}

func (p ProceedRouter) StartProceedTrack(trackInfo *pb.ProceedTrack) {
	proc, ok := p.proceedings[trackInfo.DstSessionId]
	if ok { // peer exist?
		p.ReplaceProceedTrack(proc.info, trackInfo) // if exist, just update
		return
	}
	pro := p.factory.New(proto.Clone(trackInfo).(*pb.ProceedTrack))
	proc = proceeding{
		WatchDog: util.NewWatchDog(bridge.NewBridgeFactory(p.sfu, pro)),
		info:     proto.Clone(trackInfo).(*pb.ProceedTrack),
	}
	proc.Watch(bridge.ProceedTrackParam{ProceedTrack: proto.Clone(trackInfo).(*pb.ProceedTrack)})
	p.proceedings[trackInfo.DstSessionId] = proc
}

func (p ProceedRouter) StopProceedTrack(trackInfo *pb.ProceedTrack) {
	if proc, ok := p.proceedings[trackInfo.DstSessionId]; ok { // peer exist?
		proc.Leave() // if exist, just stop
		delete(p.proceedings, trackInfo.DstSessionId)
	}
}

func (p ProceedRouter) ReplaceProceedTrack(oldTrackInfo *pb.ProceedTrack, newTrackInfo *pb.ProceedTrack) {
	if oldTrackInfo.DstSessionId != newTrackInfo.DstSessionId {
		log.Warnf("Cannot ReplaceProceedTrack when DstSessionId is not same")
		return
	}
	proc, ok := p.proceedings[oldTrackInfo.DstSessionId]
	if !ok { // peer not exist?
		return // just return
	}
	proc.Update(bridge.ProceedTrackParam{ProceedTrack: proto.Clone(newTrackInfo).(*pb.ProceedTrack)})
	proc.info = proto.Clone(newTrackInfo).(*pb.ProceedTrack)
}
