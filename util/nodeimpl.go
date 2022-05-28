package util

import (
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/protobuf/proto"
)

type DiscoveryNodeItem struct {
	Node *discovery.Node
}

func (i DiscoveryNodeItem) Key() string {
	return i.Node.NID
}
func (i DiscoveryNodeItem) Compare(data DisorderSetItem) bool {
	return i.Key() == data.Key() &&
		i.Node.DC == data.(DiscoveryNodeItem).Node.DC &&
		i.Node.Service == data.(DiscoveryNodeItem).Node.Service &&
		i.Node.RPC.Protocol == data.(DiscoveryNodeItem).Node.RPC.Protocol &&
		i.Node.RPC.Addr == data.(DiscoveryNodeItem).Node.RPC.Addr
}
func (i DiscoveryNodeItem) Clone() DisorderSetItem {
	return DiscoveryNodeItem{
		Node: &discovery.Node{
			DC:      i.Node.DC,
			Service: i.Node.Service,
			NID:     i.Node.NID,
			RPC: discovery.RPC{
				Protocol: i.Node.RPC.Protocol,
				Addr:     i.Node.RPC.Addr,
				Params:   i.Node.RPC.Params,
			},
			ExtraInfo: i.Node.ExtraInfo,
		},
	}
}

type SFUStatusItem struct {
	SFUStatus *pb.SFUStatus
}

func (i SFUStatusItem) Key() string {
	return i.SFUStatus.SFU.Nid
}
func (i SFUStatusItem) Compare(data DisorderSetItem) bool {
	// TODO: ForwardTracks, ProceedTracks and ClientNeededSession maybe disorder
	return i.SFUStatus.String() == data.(SFUStatusItem).SFUStatus.String()
}
func (i SFUStatusItem) Clone() DisorderSetItem {
	return SFUStatusItem{
		SFUStatus: proto.Clone(i.SFUStatus).(*pb.SFUStatus),
	}
}
