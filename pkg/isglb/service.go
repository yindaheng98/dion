package isglb

import (
	"fmt"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/yindaheng98/dion/util"
	"io"
	"sync"

	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/algorithms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "github.com/yindaheng98/dion/proto"
)

// ISGLBService represents isglb node
type ISGLBService struct {
	pb.UnimplementedISGLBServer
	Alg algorithms.Algorithm // The core algorithm

	recvCh   chan isglbRecvMessage
	recvChMu chan bool

	sendChs   map[*pb.ISGLB_SyncSFUServer]chan *pb.SFUStatus
	sendChsMu *sync.RWMutex
}

func NewISGLBService(alg algorithms.Algorithm) *ISGLBService {
	recvChMu := make(chan bool, 1)
	recvChMu <- true
	return &ISGLBService{
		UnimplementedISGLBServer: pb.UnimplementedISGLBServer{},
		Alg:                      alg,
		recvCh:                   make(chan isglbRecvMessage, 4096),
		recvChMu:                 recvChMu,
		sendChs:                  make(map[*pb.ISGLB_SyncSFUServer]chan *pb.SFUStatus),
		sendChsMu:                &sync.RWMutex{},
	}
}

func (isglb *ISGLBService) RegisterService(registrar grpc.ServiceRegistrar) {
	pb.RegisterISGLBServer(registrar, isglb)
}

// isglbRecvMessage represents the message flow in ISGLBService.recvCh
// the SFUStatus and a channel receive response
type isglbRecvMessage struct {
	request *pb.SyncRequest
	sigkey  *pb.ISGLB_SyncSFUServer
	deleted *discovery.Node
}

// SyncSFU receive current SFUStatus, call the algorithm, and reply expected SFUStatus
func (isglb *ISGLBService) SyncSFU(sig pb.ISGLB_SyncSFUServer) error {
	skey := &sig
	sendCh := make(chan *pb.SFUStatus)
	isglb.sendChsMu.Lock()
	isglb.sendChs[skey] = sendCh // Create send channel when begin
	isglb.sendChsMu.Unlock()
	defer func(isglb *ISGLBService, skey *pb.ISGLB_SyncSFUServer) {
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
func (isglb *ISGLBService) routineSFUStatusRecv() {
	select {
	case <-isglb.recvChMu: // If the routineSFUStatusRecv not started
		//Then start it
		defer func() { isglb.recvChMu <- true }()
	default: // If the routineSFUStatusRecv has started
		return // Do not start again
	}
	signids := make(map[string]*pb.ISGLB_SyncSFUServer)
	latestStatus := make(map[string]util.SFUStatusItem) // Just for filter out those unchanged SFUStatus
	for {
		var recvCount = 0
		savedReports := make(map[string]*pb.QualityReport) // Just for filter out those deprecated reports
	L:
		for {
			var msg isglbRecvMessage
			var ok bool
			if recvCount <= 0 { //if there is no message
				msg, ok = <-isglb.recvCh //wait for the first message
				if !ok {                 //if closed
					return //exit
				}
			} else {
				select {
				case msg, ok = <-isglb.recvCh: // Receive a message
					if !ok { //if closed
						return //exit
					}
				default: //if there is no more message
					break L //just exit
				}
			}

			if deletedNode := msg.deleted; deletedNode != nil {
				nid := (util.DiscoveryNodeItem{Node: deletedNode}).Key()
				if lastStatus, ok := latestStatus[nid]; ok {
					delete(latestStatus, nid)
					delete(signids, nid)
					log.Debugf("Deleted a SFU status: %s", lastStatus.SFUStatus.String())
					recvCount++ // count the message
				}
			}

			if msg.request == nil || msg.sigkey == nil {
				continue
			}
			//category and save messages
			switch request := msg.request.Request.(type) {
			case *pb.SyncRequest_Report:
				log.Debugf("Received a QualityReport: %s", request.Report.String())
				if _, ok = savedReports[request.Report.String()]; !ok { //filter out deprecated report
					savedReports[request.Report.String()] = proto.Clone(request.Report).(*pb.QualityReport) // Save the copy
					recvCount++                                                                             // count the message
				}
			case *pb.SyncRequest_Status:
				log.Debugf("Received a SFUStatus: %s", request.Status.String())
				reportedStatus := util.SFUStatusItem{SFUStatus: request.Status}
				nid := reportedStatus.Key()

				if lastSigkey, ok := signids[reportedStatus.Key()]; ok && lastSigkey != msg.sigkey {
					log.Warnf("Dropped deprecated SFU status sync client for node key: %s", reportedStatus.Key())
					continue
				}
				signids[nid] = msg.sigkey // Save sig and nid

				if lastStatus, ok := latestStatus[nid]; ok && lastStatus.Compare(reportedStatus) {
					log.Debugf("Dropped deprecated SFU status from request: %s", lastStatus.SFUStatus.String())
					continue //filter out unchanged status
				}
				// If the request has changed
				latestStatus[nid] = reportedStatus.Clone().(util.SFUStatusItem) // Save SFUStatus copy
				recvCount++                                                     // count the message
			}
		}

		// proceed all those received messages above
		if recvCount <= 0 { //if there is no valid message
			continue //do nothing
		}

		var i int
		statuss := make([]*pb.SFUStatus, len(latestStatus))
		i = 0
		for _, s := range latestStatus {
			statuss[i] = s.Clone().(util.SFUStatusItem).SFUStatus
			i++
		}
		i = 0
		reports := make([]*pb.QualityReport, len(savedReports))
		for _, r := range savedReports {
			reports[i] = r
			i++
		}
		expectedStatusList := isglb.Alg.UpdateSFUStatus(statuss, reports) // update algorithm
		expectedStatusDict := make(map[string]util.SFUStatusItem, len(expectedStatusList))
		for _, expectedStatus := range expectedStatusList {
			item := util.SFUStatusItem{SFUStatus: expectedStatus}
			expectedStatusDict[item.Key()] = item.Clone().(util.SFUStatusItem) // Copy the message
		}
		for nid, expectedStatus := range expectedStatusDict {
			if lastStatus, ok := latestStatus[nid]; ok && lastStatus.Compare(expectedStatus) {
				log.Debugf("Dropped deprecated SFU status from algorithm: %s", lastStatus.SFUStatus.String())
				continue //filter out unchanged request
			}
			// If the request should be change
			isglb.sendChsMu.RLock()
			if sendCh, ok := isglb.sendChs[signids[nid]]; ok {
				sendCh <- expectedStatus.SFUStatus // Send it
				latestStatus[nid] = expectedStatus // And Save it
			} else {
				log.Warnf("No SFU status sender found for nid %s: %s", nid, expectedStatus.SFUStatus.String())
			}
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
						log.Errorf("%v SFU request send error", err)
					}
				}
			}(latestStatusCh)
		}
		select {
		case latestStatusCh <- msg: //check if there is a message not send
		// no message, that's ok, our message pushed
		default: //if there is a message not send
			select {
			case <-latestStatusCh: //delete it
			default:
			}
			latestStatusCh <- msg //and push the latest message
		}
	}
}
