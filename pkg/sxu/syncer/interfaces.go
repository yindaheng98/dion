package syncer

import (
	pb "github.com/yindaheng98/isglb/proto"
)

// TrackRouter describe an abstract SFU that can route video tracks
type TrackRouter interface {
	// All these methods should be NON-BLOCK!

	// StartForwardTrack begin a track route
	StartForwardTrack(trackInfo *pb.ForwardTrack)
	// StopForwardTrack end a track route
	StopForwardTrack(trackInfo *pb.ForwardTrack)
	// StartProceedTrack begin a track proceed
	StartProceedTrack(trackInfo *pb.ProceedTrack)
	// StopProceedTrack end a track proceed
	StopProceedTrack(trackInfo *pb.ProceedTrack)
}

// QualityReporter describe an abstract SFU that can report the running quality
type QualityReporter interface {
	// FetchReport fetch a quality report
	// Block until return a new quality report
	FetchReport() *pb.QualityReport
}

type SessionEvent_State int32

const (
	SessionEvent_ADD SessionEvent_State = 0
	SessionEvent_REMOVE
)

// SessionEvent describe a event, user's join or leave
type SessionEvent struct {
	UserID    string
	SessionID string
	State     SessionEvent_State
}

// SessionTracker describe an abstract SFU that can report the user's join and leave
type SessionTracker interface {
	// FetchSessionEvent fetch a SessionEvent
	// Block until return a new SessionEvent
	FetchSessionEvent() SessionEvent
}
