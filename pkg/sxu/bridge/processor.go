package bridge

import (
	"context"
	"github.com/pion/webrtc/v3"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"google.golang.org/protobuf/proto"
)

type Processor interface {
	// AddTrack add a track to Processor
	// read video from `remote` process it and write the result to `local`
	// r/w should begin after `<-shouldBegin.Done()`
	// r/w should stop when error occurred
	AddTrack(shouldBegin context.Context, remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) (local webrtc.TrackLocal)

	// UpdateProcedure update the procedure in the Processor
	UpdateProcedure(procedure *pb.Procedure)
}

type Procedure struct {
	procedure *pb.Procedure
}

func (p Procedure) Clone() util.Param {
	return Procedure{procedure: proto.Clone(p.procedure).(*pb.Procedure)}
}
