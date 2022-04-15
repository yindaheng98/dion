package isglb

import (
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
}

// SyncSFUStatus receive current SFUStatus, call the algorithm, and reply expected SFUStatus
func (isglb *ISGLB) SyncSFUStatus(sig pb.ISGLB_SyncSFUStatusServer) error {
	skey := &sig
	isglb.sigsMu.Lock()
	isglb.sigs[skey] = true // Save the link when begin
	isglb.sigsMu.Unlock()
	defer func(isglb *ISGLB, skey *pb.ISGLB_SyncSFUStatusServer) {
		isglb.sigsMu.Lock()
		delete(isglb.sigs, skey) // delete the link when exit
		isglb.sigsMu.Unlock()
	}(isglb, skey)
	recvFinishCh := make(chan error)
	go isglb.goroutineSFUStatusRecv(sig, recvFinishCh)
	select {
	case err := <-recvFinishCh:
		if err != nil {
			return err
		}
		return nil
	}
}

func (isglb *ISGLB) goroutineSFUStatusRecv(sig pb.ISGLB_SyncSFUStatusServer, finishCh chan<- error) {
	for {
		in, err := sig.Recv() // Recv a SFUStatus
		if err != nil {
			if err == io.EOF {
				finishCh <- nil
				return
			}
			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				finishCh <- nil
				return
			}
			log.Errorf("%v SFU status receive error %d", fmt.Errorf(errStatus.Message()), errStatus.Code())
			finishCh <- err
			return
		}
		isglb.handleSFUStatus(in, sig) // handle the received SFUStatus
	}
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
