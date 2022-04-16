package isglb

import (
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/util"
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
	Alg algorithms.Algorithm // The core algorithm

	recvCh   chan isglbRecvMessage
	recvChMu chan bool

	sendChs   map[*pb.ISGLB_SyncSFUServer]chan *pb.SFUStatus
	sendChsMu *sync.RWMutex
}

func NewISGLB(alg algorithms.Algorithm) *ISGLB {
	return &ISGLB{
		UnimplementedISGLBServer: pb.UnimplementedISGLBServer{},
		Node:                     ion.NewNode("isglb-" + util.RandomString(6)),
		Alg:                      alg,
		recvCh:                   make(chan isglbRecvMessage, 1024),
		recvChMu:                 make(chan bool, 1),
		sendChs:                  make(map[*pb.ISGLB_SyncSFUServer]chan *pb.SFUStatus),
		sendChsMu:                &sync.RWMutex{},
	}
}

// isglbRecvMessage represents the message flow in ISGLB.recvCh
// the SFUStatus and a channel receive response
type isglbRecvMessage struct {
	request *pb.SyncRequest
	sigkey  *pb.ISGLB_SyncSFUServer
}

// SyncSFU receive current SFUStatus, call the algorithm, and reply expected SFUStatus
func (isglb *ISGLB) SyncSFU(sig pb.ISGLB_SyncSFUServer) error {
	skey := &sig
	sendCh := make(chan *pb.SFUStatus)
	isglb.sendChsMu.Lock()
	isglb.sendChs[skey] = sendCh // Create send channel when begin
	isglb.sendChsMu.Unlock()
	defer func(isglb *ISGLB, skey *pb.ISGLB_SyncSFUServer) {
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
		req, err := sig.Recv() // Receive a SyncRequest
		if err != nil {
			if err == io.EOF {
				return nil
			}
			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				return nil
			}
			log.Errorf("%v SFU request receive error %d", fmt.Errorf(errStatus.Message()), errStatus.Code())
			return err
		}
		// Push to receive channel
		isglb.recvCh <- isglbRecvMessage{
			request: req,
			sigkey:  &sig,
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
	signids := make(map[string]*pb.ISGLB_SyncSFUServer)
	latestStatus := make(map[string]*pb.SFUStatus) // Just for filter out those unchanged SFUStatus
	for {
		msg, ok := <-isglb.recvCh // Receive message
		if !ok {
			return
		}
		var expectedStatusList []*pb.SFUStatus
		switch request := msg.request.Request.(type) {
		case *pb.SyncRequest_Report:
			report := request.Report
			expectedStatusList = isglb.Alg.UpdateQuality(report) // update algorithm
			// TODO: report update may be fast, debounce it
		case *pb.SyncRequest_Status:
			reportedStatus := request.Status
			nid := reportedStatus.GetSFU().GetNid()
			if lastSigkey, ok := signids[nid]; ok && lastSigkey != msg.sigkey {
				log.Warnf("deprecated SFU status sync client for nid: %s", nid)
				continue
			}
			signids[nid] = msg.sigkey // Save sig and nid
			if lastStatus, ok := latestStatus[nid]; ok && lastStatus.String() == reportedStatus.String() {
				continue //filter out unchanged request
			}
			// If the request has changed
			latestStatus[nid] = reportedStatus                             // Save SFUStatus
			expectedStatusList = isglb.Alg.UpdateSFUStatus(reportedStatus) // update algorithm
			// TODO: status update may be fast, debounce it
		}
		for _, expectedStatus := range expectedStatusList {
			nid := expectedStatus.GetSFU().GetNid()
			if lastStatus, ok := latestStatus[nid]; ok && lastStatus.String() == expectedStatus.String() {
				continue //filter out unchanged request
			}
			// If the request should be change
			latestStatus[nid] = expectedStatus // Save it
			isglb.sendChsMu.RLock()
			isglb.sendChs[signids[nid]] <- expectedStatus // Send it
			isglb.sendChsMu.RUnlock()
		}
	}
}

func routineSFUStatusSend(sig pb.ISGLB_SyncSFUServer, sendCh <-chan *pb.SFUStatus) {
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
		if !ok { //If latest request not exists
			latestStatusCh = make(chan *pb.SFUStatus, 1)
			latestStatusChs[msg.GetSFU().GetNid()] = latestStatusCh //Then create it
			//and create the sender goroutine
			go func(latestStatusCh <-chan *pb.SFUStatus) {
				for {
					latestStatus, ok := <-latestStatusCh //get request
					if !ok {                             //if chan closed
						return //exit
					}
					// If the request should be change
					err := sig.Send(latestStatus)
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
