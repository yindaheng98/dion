package syncer

import (
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	pbion "github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/isglb/pkg/isglb"
	pb "github.com/yindaheng98/isglb/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"io"
)

// ISGLBSyncer is a ISGLBClient to sync SFUStatus
type ISGLBSyncer struct {
	client  *isglb.ISGLBClient
	node    *ion.Node
	descSFU *pbion.Node

	router   TrackRouter
	reporter QualityReporter
	session  SessionTracker

	clientSessionIndex Index
	forwardTrackIndex  Index
	proceedTrackIndex  Index

	// Just recv and send latest status
	statusRecvCh   chan *pb.SFUStatus
	statusSendCh   chan bool
	sessionEventCh chan *SessionEvent
}

func NewSFUStatusSyncer(node *ion.Node, peerID string, descSFU *pbion.Node, router TrackRouter, reporter QualityReporter, session SessionTracker) *ISGLBSyncer {
	isglbClient := isglb.NewISGLBClient(node, peerID, map[string]interface{}{})
	if isglbClient == nil {
		return nil
	}
	s := &ISGLBSyncer{
		client:  isglbClient,
		node:    node,
		descSFU: descSFU,

		router:   router,
		reporter: reporter,
		session:  session,

		clientSessionIndex: NewIndex(),
		forwardTrackIndex:  NewIndex(),
		proceedTrackIndex:  NewIndex(),

		statusRecvCh:   make(chan *pb.SFUStatus, 1),
		statusSendCh:   make(chan bool, 1),
		sessionEventCh: make(chan *SessionEvent, 1024),
	}
	isglbClient.OnSFUStatusRecv = func(st *pb.SFUStatus) {
		select {
		case _, ok := <-s.statusRecvCh:
			if !ok {
				return
			}
		default:
		}
		s.statusRecvCh <- st
	}
	return s
}

func (s *ISGLBSyncer) NotifySFUStatus() {
	// Only send latest status
	select {
	case s.statusSendCh <- true:
	default:
	}
}

// ↓↓↓↓↓ should access Index, so keep single thread ↓↓↓↓↓

// getSelfStatus get the current SFUStatus
// MUST be single threaded
func (s *ISGLBSyncer) getSelfStatus() *pb.SFUStatus {
	return &pb.SFUStatus{
		SFU:                 proto.Clone(s.descSFU).(*pbion.Node),
		ForwardTracks:       IndexDataList(s.forwardTrackIndex.Gather()).ToForwardTracks(),
		ProceedTracks:       IndexDataList(s.proceedTrackIndex.Gather()).ToProceedTracks(),
		ClientNeededSession: IndexDataList(s.clientSessionIndex.Gather()).ToClientSessions(),
	}
}

// syncStatus sync the current SFUStatus with the expected SFUStatus
// MUST be single threaded
func (s *ISGLBSyncer) syncStatus(expectedStatus *pb.SFUStatus) {
	if expectedStatus.SFU.String() != s.descSFU.String() { // Check if the SFU status is mine
		// If not
		log.Warnf("Received SFU status is not mine, drop it: %s", expectedStatus.SFU)
		s.NotifySFUStatus() // The server must re-consider the status for our SFU
		return              // And we should wait for the right SFU status to come
	}

	// Check if the client needed session is same
	sessionIndexDataList := clientSessions(expectedStatus.ClientNeededSession).ToIndexDataList()
	if !s.clientSessionIndex.IsSame(sessionIndexDataList) { // Check if the ClientNeededSession is same
		// If not
		log.Warnf("Received SFU status have different session list, drop it: %s", expectedStatus.ClientNeededSession)
		s.NotifySFUStatus() // The server must re-consider the status for our SFU
		return              // And we should wait for the right SFU status to come
	}

	// Perform track forward change
	forwardIndexDataList := forwardTracks(expectedStatus.ForwardTracks).ToIndexDataList()
	forwardAdd, forwardDel, forwardReplace := s.forwardTrackIndex.Update(forwardIndexDataList)
	for _, track := range forwardDel {
		s.router.StopForwardTrack(track.(forwardIndexData).forwardTrack)
	}
	for _, track := range forwardReplace {
		s.router.ReplaceForwardTrack(
			track.Old.(forwardIndexData).forwardTrack,
			track.New.(forwardIndexData).forwardTrack,
		)
	}
	for _, track := range forwardAdd {
		s.router.StartForwardTrack(track.(forwardIndexData).forwardTrack)
	}

	//Perform track proceed change
	proceedIndexDataList := proceedTracks(expectedStatus.ProceedTracks).ToIndexDataList()
	proceedAdd, proceedDel, proceedReplace := s.proceedTrackIndex.Update(proceedIndexDataList)
	for _, track := range proceedDel {
		s.router.StopProceedTrack(track.(proceedIndexData).proceedTrack)
	}
	for _, track := range proceedReplace {
		s.router.ReplaceProceedTrack(
			track.Old.(proceedIndexData).proceedTrack,
			track.New.(proceedIndexData).proceedTrack,
		)
	}
	for _, track := range proceedAdd {
		s.router.StartProceedTrack(track.(proceedIndexData).proceedTrack)
	}
}

// handleSessionEvent handle the SessionEvent
// MUST be single threaded
func (s *ISGLBSyncer) handleSessionEvent(event *SessionEvent) {
	// Just add or remove it, and sand latest status
	switch event.State {
	case SessionEvent_ADD:
		s.clientSessionIndex.Add(sessionIndexData{session: event.Session})
		s.NotifySFUStatus()
	case SessionEvent_REMOVE:
		s.clientSessionIndex.Del(sessionIndexData{session: event.Session})
		s.NotifySFUStatus()
	}
}

// main is the "main function" goroutine of the NewSFUStatusSyncer
// All the methods about Index should be here, to ensure the assess is single-threaded
func (s *ISGLBSyncer) main() {
	for {
		select {
		case event, ok := <-s.sessionEventCh: // handle an event
			if !ok {
				return
			}
			s.handleSessionEvent(event) // should access Index, so keep single thread
		case st, ok := <-s.statusRecvCh: // handle a received SFU status
			if !ok {
				return
			}
			s.syncStatus(st) // should access Index, so keep single thread
		case _, ok := <-s.statusSendCh: // handle SFU status send event
			if !ok {
				return
			}
			st := s.getSelfStatus() // should access Index, so keep single thread
			go s.send(&pb.SyncRequest{Request: &pb.SyncRequest_Status{Status: st}})
		}
	}
}

// ↑↑↑↑↑ should access Index, so keep single thread ↑↑↑↑↑

func (s *ISGLBSyncer) sessionFetcher() {
	for {
		event := s.session.FetchSessionEvent()
		if event == nil {
			return
		}
		s.sessionEventCh <- event.Clone()
	}
}

func (s *ISGLBSyncer) reportFetcher() {
	for {
		report := s.reporter.FetchReport()
		if report == nil {
			return
		}
		go s.send(&pb.SyncRequest{Request: &pb.SyncRequest_Report{Report: report}})
	}
}

func (s *ISGLBSyncer) send(r *pb.SyncRequest) {
	err := s.client.SendSyncRequest(r)
	if err != nil {
		if err == io.EOF {
			return
		}
		errStatus, _ := status.FromError(err)
		if errStatus.Code() == codes.Canceled {
			return
		}
		log.Errorf("%v SFU request send error", err)
	}
}

func (s *ISGLBSyncer) Start() {
	go s.main()
	go s.reportFetcher()
	go s.sessionFetcher()
	s.client.Connect()
}

func (s *ISGLBSyncer) Stop() {
	s.client.Close()
	close(s.statusRecvCh)
	close(s.statusSendCh)
	close(s.sessionEventCh)
}
