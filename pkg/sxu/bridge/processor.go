package bridge

import (
	"context"
	"github.com/pion/webrtc/v3"
	pb "github.com/yindaheng98/dion/proto"
)

type Processor interface {
	// AddTrack add a track to Processor
	// read video from `remote` process it and write the result to `local`
	// r/w should begin after `<-shouldBegin.Done()`
	// r/w should stop when error occurred
	AddTrack(shouldBegin context.Context, remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) (local webrtc.TrackLocal)

	// UpdateProcedure update the procedure in the Processor
	UpdateProcedure(procedure *pb.ProceedTrack)
}
