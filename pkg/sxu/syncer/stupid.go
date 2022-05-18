package syncer

import (
	log "github.com/pion/ion-log"
	pb "github.com/yindaheng98/dion/proto"
	"sync"
	"time"
)

type StupidTrackForwarder struct {
}

func (t StupidTrackForwarder) StartForwardTrack(trackInfo *pb.ForwardTrack) {
	log.Warnf("No StartForwardTrack  in toolbox | %+v\n", trackInfo)
}

func (t StupidTrackForwarder) StopForwardTrack(trackInfo *pb.ForwardTrack) {
	log.Warnf("No StopForwardTrack    in toolbox | %+v\n", trackInfo)
}

func (t StupidTrackForwarder) ReplaceForwardTrack(oldTrackInfo *pb.ForwardTrack, newTrackInfo *pb.ForwardTrack) {
	log.Warnf("No ReplaceForwardTrack in toolbox | %+v -> %+v\n", oldTrackInfo, newTrackInfo)
}

type StupidTrackProcesser struct {
}

func (t StupidTrackProcesser) StartProceedTrack(trackInfo *pb.ProceedTrack) {
	log.Warnf("No StartProceedTrack   in toolbox | %+v\n", trackInfo)
}

func (t StupidTrackProcesser) StopProceedTrack(trackInfo *pb.ProceedTrack) {
	log.Warnf("No StopProceedTrack    in toolbox | %+v\n", trackInfo)
}

func (t StupidTrackProcesser) ReplaceProceedTrack(oldTrackInfo *pb.ProceedTrack, newTrackInfo *pb.ProceedTrack) {
	log.Warnf("No ReplaceProceedTrack in toolbox | %+v -> %+v\n", oldTrackInfo, newTrackInfo)
}

var WarnDalay = 4 * time.Second

type StupidTransmissionReporter struct {
	sync.Once
}

func (d *StupidTransmissionReporter) Bind(chan<- *pb.TransmissionReport) {
	go d.Do(func() {
		log.Warnf("No TransmissionReporter in toolbox")
		<-time.After(WarnDalay)
	})
}

type StupidComputationReporter struct {
	sync.Once
}

func (d *StupidComputationReporter) Bind(chan<- *pb.ComputationReport) {
	go d.Do(func() {
		log.Warnf("No ComputationReporter in toolbox")
		<-time.After(WarnDalay)
	})
}

type StupidSessionTracker struct {
}

func (d StupidSessionTracker) FetchSessionEvent() *SessionEvent {
	log.Warnf("No FetchSessionEvent in toolbox")
	<-time.After(WarnDalay)
	return nil
}
