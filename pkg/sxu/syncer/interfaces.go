package syncer

import (
	"context"
	pb "github.com/yindaheng98/isglb/proto"
	"google.golang.org/protobuf/proto"
)

// TrackRouter describe an abstract SFU that can route video tracks
type TrackRouter interface {
	// All these methods should be NON-BLOCK!

	// StartForwardTrack begin a track route
	StartForwardTrack(trackInfo *pb.ForwardTrack)
	// StopForwardTrack end a track route
	StopForwardTrack(trackInfo *pb.ForwardTrack)
	// StartProceedTrack begin a track proceed
	StartProceedTrack(trackInfo *pb.ProceedTrack)
	// StopProceedTrack end a track proceed
	StopProceedTrack(trackInfo *pb.ProceedTrack)
}

// QualityReporter describe an abstract SFU that can report the running quality
type QualityReporter interface {
	// FetchReport fetch a quality report
	// Block until return a new quality report
	FetchReport() *pb.QualityReport
}

type ReportFetcher struct {
	Reporter QualityReporter
	reportCh chan *pb.QualityReport
	ctx      context.Context
	cancel   context.CancelFunc
}

func (f *ReportFetcher) Start() {
	f.ctx, f.cancel = context.WithCancel(context.Background())
	f.reportCh = make(chan *pb.QualityReport, 1024)
	go f.routine()
}

func (f *ReportFetcher) routine() {
	for {
		// Call FetchReport
		reportCh := make(chan *pb.QualityReport)
		go func(f *ReportFetcher, reportCh chan<- *pb.QualityReport) {
			report := f.Reporter.FetchReport()
			reportCh <- proto.Clone(report).(*pb.QualityReport)
		}(f, reportCh)

		// Wait for FetchReport return or exit
		select {
		case report := <-reportCh:
			f.reportCh <- report
		case <-f.ctx.Done():
			return
		}
	}
}

func (f *ReportFetcher) Stop() {
	f.cancel()
	close(f.reportCh)
}

func (f *ReportFetcher) Fetch() []*pb.QualityReport {
	for {
		var reports []*pb.QualityReport
	L:
		for {
			var report *pb.QualityReport
			var ok bool
			if len(reports) <= 0 { // if there is no message
				report, ok = <-f.reportCh //wait for the first message
				if !ok {                  //if closed
					return reports //exit
				}
			} else {
				select {
				case report, ok = <-f.reportCh: //Receive a message
					if !ok { //if closed
						return reports //exit
					}
				default: //if there is no more message
					break L //just exit
				}
			}
			// now we received a message
			reports = append(reports, report) //save it
		}
		if len(reports) <= 0 { //if there is no valid message
			continue //do nothing
		}
		return reports
	}
}

type SessionEvent_State int32

const (
	SessionEvent_ADD SessionEvent_State = 0
	SessionEvent_REMOVE
)

// SessionEvent describe a event, user's join or leave
type SessionEvent struct {
	UserID    string
	SessionID string
	State     SessionEvent_State
}

// SessionTracker describe an abstract SFU that can report the user's join and leave
type SessionTracker interface {
	// FetchSessionEvent fetch a SessionEvent
	// Block until return a new SessionEvent
	FetchSessionEvent() SessionEvent
}
