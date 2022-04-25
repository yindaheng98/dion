package random

import (
	"github.com/pion/ion/pkg/util"
	pb "github.com/yindaheng98/isglb/proto"
)

// RandForwardTrack Generate a ForwardTrack
func RandForwardTrack() *pb.ForwardTrack {
	return &pb.ForwardTrack{
		Src:       RandNode(util.RandomString(4)),
		SessionId: util.RandomString(8),
	}
}

// RandChangeForwardTrack change a ForwardTrack
func RandChangeForwardTrack(track *pb.ForwardTrack) {
	if RandBool() {
		track.Src = RandNode(util.RandomString(4))
	}
	if RandBool() {
		track.SessionId = util.RandomString(8)
	}
}

// RandChangeForwardTracks change a list of ForwardTrack
func RandChangeForwardTracks(tracks []*pb.ForwardTrack) []*pb.ForwardTrack {
	for _, track := range tracks {
		if RandBool() {
			RandChangeForwardTrack(track)
		}
	}
	if RandBool() {
		tracks = append(tracks, RandForwardTrack())
	}
	return tracks
}

// RandProceedTrack Generate a ProceedTrack
func RandProceedTrack() *pb.ProceedTrack {
	return &pb.ProceedTrack{
		SrcTracks:    []*pb.ForwardTrack{},
		DstSessionId: util.RandomString(4),
		Procedure:    &pb.ProceedTrack_ProcedureName{ProcedureName: util.RandomString(2)},
	}
}

// RandChangeProceedTrack change a ProceedTrack
func RandChangeProceedTrack(track *pb.ProceedTrack) {
	if RandBool() {
		track.DstSessionId = util.RandomString(4)
	}
	if RandBool() {
		track.Procedure = &pb.ProceedTrack_ProcedureName{ProcedureName: util.RandomString(2)}
	}
	if RandBool() {
		track.SrcTracks = RandChangeForwardTracks(track.SrcTracks)
	}
}

// RandChangeProceedTracks change a list of ProceedTrack
func RandChangeProceedTracks(tracks []*pb.ProceedTrack) []*pb.ProceedTrack {
	for _, track := range tracks {
		if RandBool() {
			RandChangeProceedTrack(track)
		}
	}
	if RandBool() {
		tracks = append(tracks, RandProceedTrack())
	}
	return tracks
}

type RandProceedTracks struct {
	tracks []*pb.ProceedTrack
}

func (r RandProceedTracks) RandTracks() []*pb.ProceedTrack {
	r.tracks = RandChangeProceedTracks(r.tracks)
	return r.tracks
}
