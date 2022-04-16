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
	alg algorithms.Algorithm // The core algorithm

	recvCh       chan isglbRecvMessage
	recvChMu     chan bool
	signids      map[string]*pb.ISGLB_SyncSFUStatusServer
	latestStatus map[string]*pb.SFUStatus // Just for filter out those unchanged SFUStatus

	sendChs   map[*pb.ISGLB_SyncSFUStatusServer]chan *pb.SFUStatus
	sendChsMu sync.RWMutex
}

// isglbRecvMessage represents the message flow in ISGLB.recvCh
// the SFUStatus and a channel receive response
type isglbRecvMessage struct {
	status *pb.SFUStatus
	sigkey *pb.ISGLB_SyncSFUStatusServer
}

// SyncSFUStatus receive current SFUStatus, call the algorithm, and reply expected SFUStatus
func (isglb *ISGLB) SyncSFUStatus(sig pb.ISGLB_SyncSFUStatusServer) error {
	skey := &sig
	sendCh := make(chan *pb.SFUStatus)
	isglb.sendChsMu.Lock()
	isglb.sendChs[skey] = sendCh // Create send channel when begin
	isglb.sendChsMu.Unlock()
	defer func(isglb *ISGLB, skey *pb.ISGLB_SyncSFUStatusServer) {
		isglb.sendChsMu.Lock()
		if sendCh, ok := isglb.sendChs[skey]; ok {
			close(sendCh)
			delete(isglb.sendChs, skey) // delete send channel when exit
		}
		isglb.sendChsMu.Unlock()
	}(isglb, skey)

	go routineSFUStatusSend(sig, sendCh) //start message sending
	go isglb.routineSFUStatusRecv()      //start message receiving

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
		// Push to receive channel
		isglb.recvCh <- isglbRecvMessage{
			status: in,
			sigkey: &sig,
		}
	}
}

// routineSFUStatusRecv should NOT run more than once
func (isglb *ISGLB) routineSFUStatusRecv() {
	select {
	case <-isglb.recvChMu: // If the routineSFUStatusRecv not started
		//Then start it
		defer func() { isglb.recvChMu <- true }()
	default: // If the routineSFUStatusRecv has started
		return // Do not start again
	}
	for {
		msg, ok := <-isglb.recvCh // Receive message
		if !ok {
			return
		}
		nid := msg.status.GetSFU().GetNid()
		isglb.signids[nid] = msg.sigkey // Save sig and nid
		if lastStatus, ok := isglb.latestStatus[nid]; ok && lastStatus.String() == msg.status.String() {
			continue //filter out unchanged status
		}
		// If the status has changed
		isglb.latestStatus[nid] = msg.status                        // Save SFUStatus
		expectedStatusList := isglb.alg.UpdateSFUStatus(msg.status) // update algorithm
		for _, expectedStatus := range expectedStatusList {
			nid := expectedStatus.GetSFU().GetNid()
			if lastStatus, ok := isglb.latestStatus[nid]; ok && lastStatus.String() == expectedStatus.String() {
				continue //filter out unchanged status
			}
			// If the status should be change
			isglb.latestStatus[nid] = expectedStatus // Save it
			isglb.sendChsMu.RLock()
			isglb.sendChs[isglb.signids[nid]] <- expectedStatus // Send it
			isglb.sendChsMu.RUnlock()
		}
	}
}

func routineSFUStatusSend(sig pb.ISGLB_SyncSFUStatusServer, sendCh <-chan *pb.SFUStatus) {
	latestStatusChs := make(map[string]chan *pb.SFUStatus)
	defer func(latestStatusChs map[string]chan *pb.SFUStatus) {
		for nid, ch := range latestStatusChs {
			close(ch)
			delete(latestStatusChs, nid)
		}
	}(latestStatusChs)
	for {
		msg, ok := <-sendCh
		if !ok {
			return
		}
		latestStatusCh, ok := latestStatusChs[msg.GetSFU().GetNid()]
		if !ok { //If latest status not exists
			latestStatusCh = make(chan *pb.SFUStatus, 1)
			latestStatusChs[msg.GetSFU().GetNid()] = latestStatusCh //Then create it
			//and create the sender goroutine
			go func(latestStatusCh <-chan *pb.SFUStatus) {
				for {
					latestStatus, ok := <-latestStatusCh //get status
					if !ok {                             //if chan closed
						return //exit
					}
					// If the status should be change
					err := sig.Send(latestStatus)
					if err != nil {
						if err == io.EOF {
							return
						}
						errStatus, _ := status.FromError(err)
						if errStatus.Code() == codes.Canceled {
							return
						}
						log.Errorf("%v SFU status send error", err)
					}
				}
			}(latestStatusCh)
		}
		select {
		case latestStatusCh <- msg: //check if there is a message not send
		// no message, that's ok
		default: //if there is a message not send
			select {
			case <-latestStatusCh: //delete it
				latestStatusCh <- msg //and push the latest message
			default:
				latestStatusCh <- msg
			}
		}
	}
}
