package sxu

import (
	log "github.com/pion/ion-log"
	"github.com/yindaheng98/isglb/pkg/isglb"
	pb "github.com/yindaheng98/isglb/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

type forwardedTrackTuple struct {
	checked bool
	track   *pb.ForwardTrack
}
type forwardedTrackSet map[string]*forwardedTrackTuple // set<trackId>
type forwardedTrackIndex map[string]forwardedTrackSet  // map<src.NID, set<trackId>>

// resetCheck set all the bool value in forwardedTrackIndex to be false
// so you can compare it with those in SFUStatus find which is useless
func (index forwardedTrackIndex) resetCheck() {
	for _, set := range index {
		for _, forwardTrack := range set {
			forwardTrack.checked = false
		}
	}
}

type proceededTrackTuple struct { //tuple<srcTrackId, procedure>
	checked bool
	track   *pb.ProceedTrack
}

type proceededTrackIndex map[string]*proceededTrackTuple // map<dstTrackId, tuple<srcTrackId, procedure>>

// resetCheck set all the bool value in proceededTrackIndex to be false
// so you can compare it with those in SFUStatus find which is useless
func (index proceededTrackIndex) resetCheck() {
	for _, proceedTrackTuple := range index {
		proceedTrackTuple.checked = false
	}
}

type SFU interface {
	StartForwardTrack(trackInfo *pb.ForwardTrack)
	StopForwardTrack(trackInfo *pb.ForwardTrack)
	StartProceedTrack(trackInfo *pb.ProceedTrack)
	StopProceedTrack(trackInfo *pb.ProceedTrack)
}

type SXUService struct {
	isglb.ISGLBClient
	status            *pb.SFUStatus
	sfu               SFU
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
			// Then checked if the track have already forwarded
			if forwardTrack, ok := forwardTrackSet[track.TrackId]; ok {
				// already forwarded?
				if forwardTrack.track.String() == track.String() { // check if is same track
					// is the same?
					forwardTrack.checked = true // just save the checked status
					continue                    // and go next
				} else { // if not same
					s.sfu.StopForwardTrack(forwardTrack.track) // stop the current
					s.sfu.StartForwardTrack(track)             // start the new
					// and record it
					forwardTrack.track = track
					forwardTrack.checked = true
				}
			} else {
				// not forwarded?
				//save the checked status
				forwardTrackSet[track.TrackId] = &forwardedTrackTuple{checked: true, track: track}
				s.sfu.StartForwardTrack(track) // and start forward
			}
		} else { // not exists?
			// construct and save a new checked status set for this peer
			s.forwardTrackIndex[track.Src.Nid] = map[string]*forwardedTrackTuple{track.TrackId: {checked: true, track: track}}
			s.sfu.StartForwardTrack(track) // and start forward
		}
	}

	// Find those ForwardTracks in currentStatus but not in expectedStatus, stop and delete it
	for _, forwardTrackSet := range s.forwardTrackIndex {
		for _, forwardTrackTuple := range forwardTrackSet {
			if !forwardTrackTuple.checked {
				s.sfu.StopForwardTrack(forwardTrackTuple.track)
				delete(forwardTrackSet, forwardTrackTuple.track.TrackId)
			}
		}
	}

	s.proceedTrackIndex.resetCheck()
	// Find those ProceedTracks in expectedStatus but not in currentStatus, start it
	// Find those ProceedTracks in expectedStatus and in currentStatus but not same, stop it and start new
	for _, track := range expectedStatus.ProceedTracks {
		if proceedTrackTuple, ok := s.proceedTrackIndex[track.DstTrackId]; ok { // Check if the dst track exists
			// exists?
			// Then checked if the procedure and src track is the same
			if proceedTrackTuple.track.String() == track.String() {
				//same?
				proceedTrackTuple.checked = true // save the checked status
				continue                         // If same, do nothing
			} else {
				// If not same, stop the current
				s.sfu.StopProceedTrack(proceedTrackTuple.track)
				s.sfu.StartProceedTrack(track) // and make the new
				// And update the status
				proceedTrackTuple.track = track
				proceedTrackTuple.checked = true
			}
		} else { // not exists?
			// save the status
			s.proceedTrackIndex[track.DstTrackId] = &proceededTrackTuple{track: track, checked: true}
			s.sfu.StartProceedTrack(track) // And make the new
		}
	}

	// Find those ForwardTracks in currentStatus but not in expectedStatus, stop and delete it
	for _, proceedTrackTuple := range s.proceedTrackIndex {
		if !proceedTrackTuple.checked {
			s.sfu.StopProceedTrack(proceedTrackTuple.track)
			delete(s.proceedTrackIndex, proceedTrackTuple.track.DstTrackId)
		}
	}
}

func (s *SXUService) OnSFUStatusRecv(expectedStatus *pb.SFUStatus) {
	s.syncStatus(expectedStatus)
	// TODO: Only sync latest status
}
