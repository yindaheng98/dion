package syncer

import (
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/isglb/pkg/isglb"
	pb "github.com/yindaheng98/isglb/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

// SFUStatusSyncer is a ISGLBClient to sync SFUStatus
type SFUStatusSyncer struct {
	client   isglb.ISGLBClient
	node     *ion.Node
	router   TrackRouter
	reporter QualityReporter
	session  SessionTracker

	clientSessionIndex Index
	forwardTrackIndex  Index
	proceedTrackIndex  Index
}

func NewSFUStatusSyncer(isglbClient isglb.ISGLBClient, node *ion.Node, router TrackRouter, reporter QualityReporter, session SessionTracker) *SFUStatusSyncer {
	s := &SFUStatusSyncer{
		client:   isglbClient,
		node:     node,
		router:   router,
		reporter: reporter,
		session:  session,

		clientSessionIndex: NewIndex(),
		forwardTrackIndex:  NewIndex(),
		proceedTrackIndex:  NewIndex(),
	}
	isglbClient.OnSFUStatusRecv = s.OnSFUStatusRecv
	return s
}

func (s *SFUStatusSyncer) GetSelfStatus() *pb.SFUStatus {
	return &pb.SFUStatus{
		SFU:                 s.node,
		ForwardTracks:       IndexDataList(s.forwardTrackIndex.Gather()).ToForwardTracks(),
		ProceedTracks:       IndexDataList(s.proceedTrackIndex.Gather()).ToProceedTracks(),
		ClientNeededSession: IndexDataList(s.clientSessionIndex.Gather()).ToClientSessions(),
	}
}

func (s *SFUStatusSyncer) NotifySFUStatus() {
	// TODO: Only send latest status
	err := s.client.SendSyncRequest(
		&pb.SyncRequest{
			Request: &pb.SyncRequest_Status{
				Status: s.GetSelfStatus(),
			},
		},
	)
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

// statusCheck chack whether the received expectedStatus is the same as s.status
// MUST be single threaded
func (s *SFUStatusSyncer) syncStatus(expectedStatus *pb.SFUStatus) {
	if expectedStatus.SFU.String() != s.node.String() { // Check if the SFU status is mine
		log.Warnf("Received SFU status is not mine: %s", expectedStatus.SFU) // If not
		s.NotifySFUStatus()                                                  // The server must re-consider the status for our SFU
		return                                                               // And we should wait for the right SFU status to come
	}

	// Check if the client needed session is same
	sessionIndexDataList := make([]IndexData, len(expectedStatus.ClientNeededSession))
	for i, session := range expectedStatus.ClientNeededSession {
		sessionIndexDataList[i] = sessionIndexData{session: session}
	}
	if !s.clientSessionIndex.IsSame(sessionIndexDataList) { // If not
		s.NotifySFUStatus() // The server must re-consider the status for our SFU
		return              // And we should wait for the right SFU status to come
	}

	// Perform track forward change
	forwardIndexDataList := make([]IndexData, len(expectedStatus.ForwardTracks))
	for i, track := range expectedStatus.ForwardTracks {
		forwardIndexDataList[i] = forwardIndexData{forwardTrack: track}
	}
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
	proceedIndexDataList := make([]IndexData, len(expectedStatus.ProceedTracks))
	for i, track := range expectedStatus.ProceedTracks {
		proceedIndexDataList[i] = proceedIndexData{proceedTrack: track}
	}
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

func (s *SFUStatusSyncer) OnSFUStatusRecv(expectedStatus *pb.SFUStatus) {

	s.syncStatus(expectedStatus)
	// TODO: Only sync latest status
}
