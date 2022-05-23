package algorithms

import (
	"github.com/pion/webrtc/v3"
	pb "github.com/yindaheng98/dion/proto"
)

// Processor is a processor that can process webrtc.TrackRemote and output webrtc.TrackLocal
// MULTI-THREAD access!!! Should implemented in THREAD-SAFE!!!
type Processor interface {

	// Init set the AddTrack, RemoveTrack and OnBroken func and init the output track from Processor
	// Should be NON-BLOCK!
	// after you created a new track, please call AddTrack
	// before you close a track, please call RemoveTrack
	// when occurred error, please call OnBroken
	Init(
		AddTrack func(webrtc.TrackLocal) (*webrtc.RTPSender, error),
		RemoveTrack func(*webrtc.RTPSender) error,
		OnBroken func(badGay error),
	) error

	// AddInTrack add a input track to Processor
	// Will be called AFTER InitOutTrack!
	// read video from `remote` process it and write the result to the output track
	// r/w should stop when error occurred
	// Should be NON-BLOCK!
	AddInTrack(SID string, remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) error

	// UpdateProcedure update the procedure in the Processor
	UpdateProcedure(procedure *pb.ProceedTrack) error
}

type ProcessorFactory interface {
	NewProcessor() (Processor, error)
}
