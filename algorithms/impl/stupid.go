package impl

import (
	"github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
)

type StupidAlgorithm struct {
}

func (StupidAlgorithm) UpdateSFUStatus(current []*pb.SFUStatus, reports []*pb.QualityReport) (expected []*pb.SFUStatus) {
L:
	for _, n := range current {
		for _, track := range n.ForwardTracks {
			if track.Src.Service == config.ServiceStupid &&
				track.Src.Nid == config.ServiceNameStupid &&
				track.RemoteSessionId == config.ServiceSessionStupid {
				continue L
			}
		}
		n.ForwardTracks = append(n.ForwardTracks, &pb.ForwardTrack{
			Src: &ion.Node{
				Nid:     config.ServiceNameStupid,
				Service: config.ServiceStupid,
			},
			RemoteSessionId: config.ServiceSessionStupid,
			LocalSessionId:  config.ServiceSessionStupid,
		})
	}
	return current
}
