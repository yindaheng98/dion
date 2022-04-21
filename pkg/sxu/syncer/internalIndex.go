package syncer

import (
	pb "github.com/yindaheng98/isglb/proto"
	"google.golang.org/protobuf/proto"
)

type sessionIndexData struct {
	session *pb.ClientNeededSession
}

func (d sessionIndexData) Key() string {
	return d.session.User + d.session.Session
}
func (d sessionIndexData) Compare(data IndexData) bool {
	return d.session.String() == data.(sessionIndexData).session.String()
}
func (d sessionIndexData) Clone() IndexData {
	return sessionIndexData{
		session: proto.Clone(d.session).(*pb.ClientNeededSession),
	}
}

type forwardIndexData struct {
	forwardTrack *pb.ForwardTrack
}

func (d forwardIndexData) Key() string {
	return d.forwardTrack.Src.Nid + d.forwardTrack.TrackId
}
func (d forwardIndexData) Compare(data IndexData) bool {
	return d.forwardTrack.String() == data.(forwardIndexData).forwardTrack.String()
}
func (d forwardIndexData) Clone() IndexData {
	return forwardIndexData{
		forwardTrack: proto.Clone(d.forwardTrack).(*pb.ForwardTrack),
	}
}

type proceedIndexData struct {
	proceedTrack *pb.ProceedTrack
}

func (d proceedIndexData) Key() string {
	return d.proceedTrack.DstTrackId
}
func (d proceedIndexData) Compare(data IndexData) bool {
	return d.proceedTrack.String() == data.(proceedIndexData).proceedTrack.String()
}
func (d proceedIndexData) Clone() IndexData {
	return proceedIndexData{
		proceedTrack: proto.Clone(d.proceedTrack).(*pb.ProceedTrack),
	}
}

type IndexDataList []IndexData

func (list IndexDataList) ToClientSessions() []*pb.ClientNeededSession {
	tracks := make([]*pb.ClientNeededSession, len(list))
	for i, data := range list {
		tracks[i] = data.(sessionIndexData).session
	}
	return tracks
}

func (list IndexDataList) ToForwardTracks() []*pb.ForwardTrack {
	tracks := make([]*pb.ForwardTrack, len(list))
	for i, data := range list {
		tracks[i] = data.(forwardIndexData).forwardTrack
	}
	return tracks
}

func (list IndexDataList) ToProceedTracks() []*pb.ProceedTrack {
	tracks := make([]*pb.ProceedTrack, len(list))
	for i, data := range list {
		tracks[i] = data.(proceedIndexData).proceedTrack
	}
	return tracks
}
