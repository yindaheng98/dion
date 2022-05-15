package bridge

import (
	"github.com/pion/webrtc/v3"
	pb "github.com/yindaheng98/dion/proto"
)

// Processor is a processor that can process webrtc.TrackRemote and output webrtc.TrackLocal
// MULTI-THREAD access!!! Should implemented in THREAD-SAFE!!!
type Processor interface {
	// AddTrack add a track to Processor
	// read video from `remote` process it and write the result to `local`
	// r/w should begin after `<-shouldBegin.Done()`
	// r/w should stop when error occurred
	// Should be NON-BLOCK!
	AddTrack(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, OnBroken func(badGay error)) (local webrtc.TrackLocal)

	// UpdateProcedure update the procedure in the Processor
	UpdateProcedure(procedure *pb.ProceedTrack)
}
