package syncer

import (
	pb "github.com/yindaheng98/isglb/proto"
)

// TODO: Keep Order

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

// construct build a []*pb.ForwardTrack for SFUStatus
func (index forwardedTrackIndex) construct() []*pb.ForwardTrack {
	total := 0
	for _, set := range index {
		for range set {
			total += 1
		}
	}

	forwardTracks := make([]*pb.ForwardTrack, total)
	i := 0
	for _, set := range index {
		for _, forwardTrack := range set {
			forwardTracks[i] = forwardTrack.track
			i += 1
		}
	}
	return forwardTracks
}

type proceededTrackTuple struct {
	checked bool
	track   *pb.ProceedTrack
}
type proceededTrackIndex map[string]*proceededTrackTuple // map<dstTrackId, track>

// resetCheck set all the bool value in proceededTrackIndex to be false
// so you can compare it with those in SFUStatus find which is useless
func (index proceededTrackIndex) resetCheck() {
	for _, proceedTrackTuple := range index {
		proceedTrackTuple.checked = false
	}
}

// construct build a []*pb.ProceedTrack for SFUStatus
func (index proceededTrackIndex) construct() []*pb.ProceedTrack {
	var proceedTracks []*pb.ProceedTrack
	for _, proceedTrackTuple := range index {
		proceedTracks = append(proceedTracks, proceedTrackTuple.track)
	}
	return proceedTracks
}

type clientSideSessionTuple struct { //tuple<ClientNeededSession.String(), ClientNeededSession>
	checked bool
	session *pb.ClientNeededSession
}
type clientSideSessionIndex map[string]clientSideSessionTuple

func (index clientSideSessionIndex) resetCheck() {
	for _, clientSessionTuple := range index {
		clientSessionTuple.checked = false
	}
}

func (index clientSideSessionIndex) construct() []*pb.ClientNeededSession {
	var clientSessions []*pb.ClientNeededSession
	for _, clientSessionTuple := range index {
		clientSessions = append(clientSessions, clientSessionTuple.session)
	}
	return clientSessions
}
