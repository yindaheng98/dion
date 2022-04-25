package random

import (
	"github.com/pion/ion/pkg/util"
	pb "github.com/yindaheng98/isglb/proto"
)

func RandForwardTrack() *pb.ForwardTrack {
	return &pb.ForwardTrack{
		Src:     RandNode(util.RandomString(4)),
		TrackId: util.RandomString(8),
	}
}

func RandChangeForwardTrack(track *pb.ForwardTrack) {
	if RandBool() {
		track.Src = RandNode(util.RandomString(4))
	}
	if RandBool() {
		track.TrackId = util.RandomString(8)
	}
}

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

func RandProceedTrack() *pb.ProceedTrack {
	return &pb.ProceedTrack{
		SrcTrackId: util.RandomString(3),
		DstTrackId: util.RandomString(4),
		Procedure:  util.RandomString(2),
	}
}

func RandChangeProceedTrack(track *pb.ProceedTrack) {
	if RandBool() {
		track.SrcTrackId = util.RandomString(3)
	}
	if RandBool() {
		track.DstTrackId = util.RandomString(4)
	}
	if RandBool() {
		track.Procedure = util.RandomString(2)
	}
}

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

type RandForwardTracks struct {
	tracks []*pb.ForwardTrack
}

func (r RandForwardTracks) RandTracks() []*pb.ForwardTrack {
	r.tracks = RandChangeForwardTracks(r.tracks)
	return r.tracks
}

type RandProceedTracks struct {
	tracks []*pb.ProceedTrack
}

func (r RandProceedTracks) RandTracks() []*pb.ProceedTrack {
	r.tracks = RandChangeProceedTracks(r.tracks)
	return r.tracks
}
