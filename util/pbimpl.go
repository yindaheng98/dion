package util

import (
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/protobuf/proto"
)

type ClientNeededSessionItem struct {
	Client *pb.ClientNeededSession
}

func (i ClientNeededSessionItem) Key() string {
	return i.Client.User + i.Client.Session
}
func (i ClientNeededSessionItem) Compare(data DisorderSetItem) bool {
	return i.Client.String() == data.(ClientNeededSessionItem).Client.String()
}
func (i ClientNeededSessionItem) Clone() DisorderSetItem {
	return ClientNeededSessionItem{
		Client: proto.Clone(i.Client).(*pb.ClientNeededSession),
	}
}

type ClientNeededSessions []*pb.ClientNeededSession

func (clients ClientNeededSessions) ToDisorderSetItemList() DisorderSetItemList {
	list := make([]DisorderSetItem, len(clients))
	for i, client := range clients {
		list[i] = ClientNeededSessionItem{Client: client}
	}
	return list
}

type ForwardTrackItem struct {
	Track *pb.ForwardTrack
}

func (i ForwardTrackItem) Key() string {
	return i.Track.Src.Nid + i.Track.RemoteSessionId // !!!重要!!!不允许多次转发同一个节点的同一个Session
}

func (i ForwardTrackItem) Compare(data DisorderSetItem) bool {
	return i.Track.String() == data.(ForwardTrackItem).Track.String()
}

func (i ForwardTrackItem) Clone() DisorderSetItem {
	return ForwardTrackItem{
		Track: proto.Clone(i.Track).(*pb.ForwardTrack),
	}
}

type ForwardTracks []*pb.ForwardTrack

func (tracks ForwardTracks) ToDisorderSetItemList() DisorderSetItemList {
	list := make([]DisorderSetItem, len(tracks))
	for i, track := range tracks {
		list[i] = ForwardTrackItem{Track: track}
	}
	return list
}

type ProceedTrackItem struct {
	Track *pb.ProceedTrack
}

func (i ProceedTrackItem) Key() string {
	return i.Track.DstSessionId // !!!重要!!!不允许多个处理结果放进一个Session里
}
func (i ProceedTrackItem) Compare(data DisorderSetItem) bool {
	srcTrackList1 := Strings(data.(ProceedTrackItem).Track.SrcSessionIdList).ToDisorderSetItemList()
	srcTrackSet1 := NewDisorderSetFromList(srcTrackList1)
	srcTrackList2 := Strings(i.Track.SrcSessionIdList).ToDisorderSetItemList()
	if !srcTrackSet1.IsSame(srcTrackList2) {
		return false
	}
	return i.Track.String() == data.(ProceedTrackItem).Track.String()
}
func (i ProceedTrackItem) Clone() DisorderSetItem {
	return ProceedTrackItem{
		Track: proto.Clone(i.Track).(*pb.ProceedTrack),
	}
}

type ProceedTracks []*pb.ProceedTrack

func (tracks ProceedTracks) ToDisorderSetItemList() DisorderSetItemList {
	indexDataList := make([]DisorderSetItem, len(tracks))
	for i, track := range tracks {
		indexDataList[i] = ProceedTrackItem{Track: track}
	}
	return indexDataList
}

type ItemList DisorderSetItemList

func (list ItemList) ToClientSessions() []*pb.ClientNeededSession {
	tracks := make([]*pb.ClientNeededSession, len(list))
	for i, data := range list {
		tracks[i] = data.(ClientNeededSessionItem).Client
	}
	return tracks
}

func (list ItemList) ToForwardTracks() []*pb.ForwardTrack {
	tracks := make([]*pb.ForwardTrack, len(list))
	for i, data := range list {
		tracks[i] = data.(ForwardTrackItem).Track
	}
	return tracks
}

func (list ItemList) ToProceedTracks() []*pb.ProceedTrack {
	tracks := make([]*pb.ProceedTrack, len(list))
	for i, data := range list {
		tracks[i] = data.(ProceedTrackItem).Track
	}
	return tracks
}
