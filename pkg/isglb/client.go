package isglb

import (
	"context"
	"fmt"
	"github.com/yindaheng98/dion/config"
	"io"

	log "github.com/pion/ion-log"
	sdk "github.com/pion/ion-sdk-go"
	"github.com/pion/ion/pkg/ion"
	pb "github.com/yindaheng98/dion/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ISGLBClient struct {
	sdk.Service
	connected   bool
	isglbClient pb.ISGLBClient
	isglbStream pb.ISGLB_SyncSFUClient

	ctx    context.Context
	cancel context.CancelFunc

	Metadata metadata.MD
	grpcConn grpc.ClientConnInterface

	OnSFUStatusRecv func(s *pb.SFUStatus)
}

func NewISGLBClient(node *ion.Node, peerNID string, parameters map[string]interface{}) *ISGLBClient {
	ncli, err := node.NewNatsRPCClient(config.ServiceISGLB, peerNID, parameters)
	if err != nil {
		log.Errorf("error: %v", err)
		return nil
	}
	c := &ISGLBClient{
		grpcConn: ncli,
	}
	return c
}

func (c *ISGLBClient) SendSyncRequest(r *pb.SyncRequest) error {
	err := c.isglbStream.Send(r)
	if err != nil {
		return err
	}
	return nil
}

func (c *ISGLBClient) isglbReadLoop(cancelFunc context.CancelFunc) {
	for {
		s, err := c.isglbStream.Recv() // Receive a SyncRequest
		if err != nil {
			if err == io.EOF {
				cancelFunc()
				return
			}
			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				cancelFunc()
				return
			}
			log.Errorf("%v SFU status receive error %d", fmt.Errorf(errStatus.Message()), errStatus.Code())
			cancelFunc()
			return
		}
		c.OnSFUStatusRecv(s)
	}
}

// ↓↓↓↓↓ COPY FROM https://github.com/pion/ion-sdk-go/blob/34fbe58bfa24a62a2ced716c2608a0a9fb2dbae0/room.go ↓↓↓↓↓

func (c *ISGLBClient) Close() {
	c.cancel()
	_ = c.isglbStream.CloseSend()
	log.Infof("Close ok")
}

func (c *ISGLBClient) Name() string {
	return "ISGLBClient"
}

func (c *ISGLBClient) Connect() {
	var err error
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.ctx = metadata.NewOutgoingContext(c.ctx, c.Metadata)
	c.isglbClient = pb.NewISGLBClient(c.grpcConn)
	c.isglbStream, err = c.isglbClient.SyncSFU(c.ctx)

	if err != nil {
		log.Errorf("error: %v", err)
		return
	}
	go c.isglbReadLoop(c.cancel)
	c.connected = true
	log.Infof("ISGLBClient.Connect!")
}

func (c *ISGLBClient) Connected() bool {
	return c.connected
}

// ↑↑↑↑↑ COPY FROM https://github.com/pion/ion-sdk-go/blob/34fbe58bfa24a62a2ced716c2608a0a9fb2dbae0/room.go ↑↑↑↑↑
