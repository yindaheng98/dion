package isglb

import (
	"context"
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/ion"
	"github.com/yindaheng98/isglb/algorithms"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"sync"
)
import pb "github.com/yindaheng98/isglb/proto"

// ISGLB represents isglb node
type ISGLB struct {
	pb.UnimplementedISGLBServer
	ion.Node
	sigs   map[*pb.ISGLB_SyncSFUStatusServer]bool
	sigsMu sync.RWMutex
	alg    algorithms.Algorithm     // The core algorithm
	sss    map[string]*pb.SFUStatus // Just for filter out those unchanged SFUStatus
	sssMu  sync.RWMutex

	ctx     context.Context
	recvCh  chan isglbRecvMessage
	sendChs map[*pb.ISGLB_SyncSFUStatusServer]chan *pb.SFUStatus
}

// isglbRecvMessage represents the message flow in ISGLB.recvCh
// the SFUStatus and a channel receive response
type isglbRecvMessage struct {
	status *pb.SFUStatus
	respCh chan<- *pb.SFUStatus
}

// SyncSFUStatus receive current SFUStatus, call the algorithm, and reply expected SFUStatus
func (isglb *ISGLB) SyncSFUStatus(sig pb.ISGLB_SyncSFUStatusServer) error {
	skey := &sig
	sendCh := make(chan *pb.SFUStatus)
	isglb.sigsMu.Lock()
	isglb.sigs[skey] = true      // Save sig when begin
	isglb.sendChs[skey] = sendCh // Create send channel when begin
	isglb.sigsMu.Unlock()
	defer func(isglb *ISGLB, skey *pb.ISGLB_SyncSFUStatusServer) {
		isglb.sigsMu.Lock()
		delete(isglb.sigs, skey) // delete sig when exit
		if sendCh, ok := isglb.sendChs[skey]; ok {
			close(sendCh)
			delete(isglb.sendChs, skey) // delete send channel when exit
		}
		isglb.sigsMu.Unlock()
	}(isglb, skey)
	for {
		in, err := sig.Recv() // Receive a SFUStatus
		if err != nil {
			if err == io.EOF {
				return nil
			}
			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				return nil
			}
			log.Errorf("%v SFU status receive error %d", fmt.Errorf(errStatus.Message()), errStatus.Code())
			return err
		}
		isglb.recvCh <- isglbRecvMessage{
			status: in,
			respCh: sendCh,
		}
		return nil
	}
}

// routineSFUStatusRecv should NOT run more than once
func (isglb *ISGLB) routineSFUStatusRecv() {
}

func (isglb *ISGLB) routineSFUStatusSend() {
}
func (isglb *ISGLB) handleSFUStatus(ss *pb.SFUStatus, sig pb.ISGLB_SyncSFUStatusServer) {
	nid := ss.GetSFU().GetNid()
	hasReplied := false
	isglb.sssMu.Lock()
	defer func() {
		if !hasReplied { // If has not reply
			err := sig.Send(isglb.alg.GetSFUStatus(nid)) // Then reply it
			if err != nil {
				log.Errorf("OnIceCandidate send error: %v", err)
			}
		}
		isglb.sssMu.Unlock()
	}()
	lastSs, ok := isglb.sss[nid]              // Search the SFU
	if ok || lastSs.String() == ss.String() { // If exists and is the same
		return // Then just do nothing
	}
	// If not, should make a change
	isglb.sss[nid] = ss                            // Then save it
	changedStatus := isglb.alg.UpdateSFUStatus(ss) // And update the algorithm

	return
}
