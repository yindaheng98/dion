package sxu

import (
	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/pkg/sfu"
	"github.com/yindaheng98/isglb/pkg/isglb"
	pb "github.com/yindaheng98/isglb/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

type forwardedTrackSet map[string]bool                // set<trackId>
type forwardedTrackIndex map[string]forwardedTrackSet // map<src.NID, set<trackId>>

// resetCheck set all the bool value in forwardedTrackIndex to be false
// so you can compare it with those in SFUStatus find which is useless
func (index forwardedTrackIndex) resetCheck() {
	for _, set := range index {
		for trackId := range set {
			set[trackId] = false
		}
	}
}

type proceededTrackTuple struct { //tuple<srcTrackId, procedure>
	srcTrackId string
	procedure  string
	check      bool
}

type proceededTrackIndex map[string]*proceededTrackTuple // map<dstTrackId, tuple<srcTrackId, procedure>>

// resetCheck set all the bool value in proceededTrackIndex to be false
// so you can compare it with those in SFUStatus find which is useless
func (index proceededTrackIndex) resetCheck() {
	for _, proceedTrackTuple := range index {
		proceedTrackTuple.check = false
	}
}

type SXUService struct {
	isglb.ISGLBClient
	status            *pb.SFUStatus
	sfu               sfu.SFU
	forwardTrackIndex forwardedTrackIndex
	proceedTrackIndex proceededTrackIndex // map<dstTrackId, tuple<srcTrackId, procedure>>
}

func (s *SXUService) notifySFUStatus() {
	err := s.SendSyncRequest(&pb.SyncRequest{Request: &pb.SyncRequest_Status{Status: s.status}})
	if err != nil {
		if err == io.EOF {
			return
		}
		errStatus, _ := status.FromError(err)
		if errStatus.Code() == codes.Canceled {
			return
		}
		log.Errorf("%v SFU request send error", err)
	}
	// TODO: Only send latest status
}

func (s *SXUService) startForwardTrack(trackInfo *pb.ForwardTrack) {

}

func (s *SXUService) stopForwardTrack(nid, srcTrackId string) {

}

func (s *SXUService) startProceedTrack(trackInfo *pb.ProceedTrack) {

}

func (s *SXUService) stopProceedTrack(dstTrackId string) {

}

// statusCheck chack whether the received expectedStatus is the same as s.status
// MUST be single threaded
func (s *SXUService) syncStatus(expectedStatus *pb.SFUStatus) {
	selfStatus := s.status

	for i, sid := range expectedStatus.ClientNeededSession { // Check if the client needed session is same
		if selfStatus.ClientNeededSession[i] != sid { // If not
			s.notifySFUStatus() // The server must re-consider the status for our SFU
			return              // And we should wait for the right SFU status to come
		}
	}

	s.forwardTrackIndex.resetCheck()
	// Find those ForwardTracks in expectedStatus but not in currentStatus, start it
	for _, track := range expectedStatus.ForwardTracks {
		if forwardTrackSet, ok := s.forwardTrackIndex[track.Src.Nid]; ok { // Check if the peer id exists
			// exists?
			if _, ok = forwardTrackSet[track.TrackId]; ok { // Then check if the track have already forwarded
				// already forwarded?
				forwardTrackSet[track.TrackId] = true // just save the check status
				continue                              // and go next
			} else {
				// not forwarded?
				forwardTrackSet[track.TrackId] = true // save the check status
				s.startForwardTrack(track)            // and start forward
			}
		} else { // not exists?
			// construct and save a new check status set for this peer
			s.forwardTrackIndex[track.Src.Nid] = map[string]bool{track.TrackId: true}
			s.startForwardTrack(track) // and start forward
		}
	}

	// Find those ForwardTracks in currentStatus but not in expectedStatus, stop it
	for nid, forwardTrackSet := range s.forwardTrackIndex {
		for srcTrackId, shouldKeep := range forwardTrackSet {
			if !shouldKeep {
				s.stopForwardTrack(nid, srcTrackId)
			}
		}
	}

	s.proceedTrackIndex.resetCheck()
	// Find those ProceedTracks in expectedStatus but not in currentStatus, start it
	// Find those ProceedTracks in expectedStatus and in currentStatus but not same, stop it and start new
	for _, track := range expectedStatus.ProceedTracks {
		if proceedTrackTuple, ok := s.proceedTrackIndex[track.DstTrackId]; ok { // Check if the dst track exists
			// exists?
			proceedTrackTuple.check = true // save the check status
			// Then check if the procedure and src track is the same
			if proceedTrackTuple.procedure == track.Procedure && proceedTrackTuple.srcTrackId == track.SrcTrackId {
				continue // If same, do nothing
			} else {
				// If not same, stop the current
				s.stopProceedTrack(track.DstTrackId)
				// And update the status
				proceedTrackTuple.procedure = track.Procedure
				proceedTrackTuple.srcTrackId = track.SrcTrackId
				// and make the new
				s.startProceedTrack(track)
			}
		} else { // not exists?
			// save the status
			s.proceedTrackIndex[track.DstTrackId] = &proceededTrackTuple{
				srcTrackId: track.SrcTrackId,
				procedure:  track.Procedure,
				check:      true,
			}
			s.startProceedTrack(track) // And make the new
		}
	}

	// Find those ForwardTracks in currentStatus but not in expectedStatus, stop it
	for dstTrackId, proceedTrackTuple := range s.proceedTrackIndex {
		if !proceedTrackTuple.check {
			s.stopProceedTrack(dstTrackId)
		}
	}
}

func (s *SXUService) OnSFUStatusRecv(expectedStatus *pb.SFUStatus) {
	s.syncStatus(expectedStatus)
	// TODO: Only sync latest status
}
