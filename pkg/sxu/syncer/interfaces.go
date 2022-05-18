package syncer

import (
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/protobuf/proto"
)

type ToolBox struct {
	TrackForwarder       TrackForwarder
	TrackProcessor       TrackProcessor
	TransmissionReporter TransmissionReporter
	ComputationReporter  ComputationReporter
	SessionTracker       SessionTracker
}

// trackRouter describe an abstract SFU that can route video tracks
type trackRouter struct {
	// All these methods should be NON-BLOCK!

	TrackForwarder
	TrackProcessor
}

// TrackForwarder describe an abstract SFU that can forward video tracks
type TrackForwarder interface {
	// All these methods should be NON-BLOCK!

	// StartForwardTrack begin a track route
	StartForwardTrack(trackInfo *pb.ForwardTrack)
	// StopForwardTrack end a track route
	StopForwardTrack(trackInfo *pb.ForwardTrack)
	// ReplaceForwardTrack change a track route
	ReplaceForwardTrack(oldTrackInfo *pb.ForwardTrack, newTrackInfo *pb.ForwardTrack)
}

// TrackProcessor describe an abstract SFU that can proceed video tracks
type TrackProcessor interface {
	// All these methods should be NON-BLOCK!

	// StartProceedTrack begin a track proceed
	StartProceedTrack(trackInfo *pb.ProceedTrack)
	// StopProceedTrack end a track proceed
	StopProceedTrack(trackInfo *pb.ProceedTrack)
	// ReplaceProceedTrack change a track proceed
	ReplaceProceedTrack(oldTrackInfo *pb.ProceedTrack, newTrackInfo *pb.ProceedTrack)
}

// TransmissionReporter describe an abstract SFU that can report the Transmission quality
type TransmissionReporter interface {
	// Bind TransmissionReporter to a channel
	// you should push your TransmissionReport into this channel
	Bind(chan<- *pb.TransmissionReport)
}

// ComputationReporter describe an abstract SFU that can report the Transmission quality
type ComputationReporter interface {
	// Bind ComputationReporter to a channel
	// you should push your ComputationReport into this channel
	Bind(chan<- *pb.ComputationReport)
}

type SessionEvent_State int32

const (
	SessionEvent_ADD    SessionEvent_State = 0
	SessionEvent_REMOVE SessionEvent_State = 1
)

// SessionEvent describe a event, user's join or leave
type SessionEvent struct {
	Session *pb.ClientNeededSession
	State   SessionEvent_State
}

// SessionTracker describe an abstract SFU that can report the user's join and leave
type SessionTracker interface {
	// FetchSessionEvent fetch a SessionEvent
	// Block until return a new SessionEvent
	FetchSessionEvent() *SessionEvent
}

func (event *SessionEvent) Clone() *SessionEvent {
	return &SessionEvent{
		Session: proto.Clone(event.Session).(*pb.ClientNeededSession),
		State:   event.State,
	}
}
