package isglb

import (
	"fmt"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/ion"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)
import pb "github.com/yindaheng98/isglb/proto"

// ISGLB represents isglb node
type ISGLB struct {
	pb.UnimplementedISGLBServer
	ion.Node
	listSFUStatus *pb.SFUStatus
}

// SyncSFUStatus receive current SFUStatus, call the algorithm, and reply expected SFUStatus
func (isglb *ISGLB) SyncSFUStatus(sig pb.ISGLB_SyncSFUStatusServer) error {
	recvFinishCh := make(chan error)
	go isglb.goroutineSFUStatusRecv(sig, recvFinishCh)
	select {
	case recvFinish := <-recvFinishCh:
		if recvFinish != nil {
			return recvFinish
		}
		return nil
	}
}

func (isglb *ISGLB) goroutineSFUStatusRecv(sig pb.ISGLB_SyncSFUStatusServer, finishCh chan<- error) {
	for { //Multi thread recv and send
		in, err := sig.Recv()
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
			log.Errorf("%v signal error %d", fmt.Errorf(errStatus.Message()), errStatus.Code())
			finishCh <- err
			return
		}
		isglb.handleSFUStatus(in) // handle the received SFUStatus
	}
}

func (*ISGLB) handleSFUStatus(ss *pb.SFUStatus) {

}
