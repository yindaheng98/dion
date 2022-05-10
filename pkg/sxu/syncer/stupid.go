package syncer

import (
	log "github.com/pion/ion-log"
	pb "github.com/yindaheng98/dion/proto"
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

type StupidQualityReporter struct {
}

func (d StupidQualityReporter) FetchReport() *pb.QualityReport {
	log.Warnf("No FetchReport in toolbox")
	<-time.After(WarnDalay)
	return nil
}

type StupidSessionTracker struct {
}

func (d StupidSessionTracker) FetchSessionEvent() *SessionEvent {
	log.Warnf("No FetchSessionEvent in toolbox")
	<-time.After(WarnDalay)
	return nil
}
