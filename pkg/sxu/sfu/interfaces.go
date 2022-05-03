package signal

import (
	log "github.com/pion/ion-log"
	sdk "github.com/pion/ion-sdk-go"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	pb "github.com/yindaheng98/dion/proto"
)

func NewIONSFU(conf ion_sfu.Config) *ion_sfu.SFU {
	return ion_sfu.NewSFU(conf)
}

type SFUTrackRouter struct {
	node              *ion.Node
	connectors        map[string]*sdk.Connector
	sfu               *ion_sfu.SFU
	forwardController ForwardController
	proceedController ProceedController
}

// TODO: All the methods should retry when failed until success

func (s *SFUTrackRouter) StartForwardTrack(trackInfo *pb.ForwardTrack) {
	err := s.forwardController.StartTransmit(trackInfo)
	if err != nil {
		log.Errorf("negotiation error: %v", err)
	}
	// TODO: Infinity retry until call StopForwardTrack or ReplaceForwardTrack
}

func (s *SFUTrackRouter) StopForwardTrack(trackInfo *pb.ForwardTrack) {
	panic("implement me")
}

func (s *SFUTrackRouter) ReplaceForwardTrack(oldTrackInfo *pb.ForwardTrack, newTrackInfo *pb.ForwardTrack) {
	panic("implement me")
}

func (s *SFUTrackRouter) StartProceedTrack(trackInfo *pb.ProceedTrack) {
	panic("implement me")
}

func (s *SFUTrackRouter) StopProceedTrack(trackInfo *pb.ProceedTrack) {
	panic("implement me")
}

func (s *SFUTrackRouter) ReplaceProceedTrack(oldTrackInfo *pb.ProceedTrack, newTrackInfo *pb.ProceedTrack) {
	panic("implement me")
}
