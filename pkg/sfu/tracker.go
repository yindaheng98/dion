package sfu

import (
	"github.com/yindaheng98/dion/pkg/sxu/syncer"
	pb "github.com/yindaheng98/dion/proto"
)

func (s *SFUService) addClient(sess *pb.ClientNeededSession) {
	s.sessionEv <- &syncer.SessionEvent{Session: sess, State: syncer.SessionEvent_ADD}
}

func (s *SFUService) removeClient(sess *pb.ClientNeededSession) {
	s.sessionEv <- &syncer.SessionEvent{Session: sess, State: syncer.SessionEvent_REMOVE}
}

func (s *SFUService) FetchSessionEvent() *syncer.SessionEvent {
	return <-s.sessionEv
}
