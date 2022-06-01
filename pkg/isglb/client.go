package isglb

import (
	"context"
	log "github.com/pion/ion-log"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"google.golang.org/grpc/metadata"
)

type ISGLBClientStreamFactory struct {
	node       *ion.Node
	peerNID    string
	parameters map[string]interface{}
	Metadata   metadata.MD
}

func (c ISGLBClientStreamFactory) NewClientStream(ctx context.Context) (util.ClientStream[*pb.SyncRequest, *pb.SFUStatus], error) {
	conn, err := c.node.NewNatsRPCClient(config.ServiceISGLB, c.peerNID, c.parameters)
	if err != nil {
		log.Errorf("cannot node.NewNatsRPCClient: %v", err)
		return nil, err
	}
	ctx = metadata.NewOutgoingContext(ctx, c.Metadata)
	client, err := pb.NewISGLBClient(conn).SyncSFU(ctx)
	if err != nil {
		log.Errorf("cannot pb.NewISGLBClient: %v", err)
		return nil, err
	}
	return client, err
}

type ISGLBClient struct {
	*util.Client[*pb.SyncRequest, *pb.SFUStatus]
	ctxTop    context.Context
	cancelTop context.CancelFunc

	sendSFUStatusExec *util.SingleLatestExec

	client     pb.ISGLB_SyncSFUClient
	cancelLast context.CancelFunc

	OnSFUStatusRecv func(s *pb.SFUStatus)
}

func NewISGLBClient(node *ion.Node, peerNID string, parameters map[string]interface{}) *ISGLBClient {
	ctx, cancal := context.WithCancel(context.Background())
	c := &ISGLBClient{
		Client: util.NewClient[*pb.SyncRequest, *pb.SFUStatus](
			ISGLBClientStreamFactory{
				node: node, peerNID: peerNID, parameters: parameters,
			}),
		ctxTop:            ctx,
		cancelTop:         cancal,
		sendSFUStatusExec: &util.SingleLatestExec{},
	}
	c.OnMsgRecv = func(status *pb.SFUStatus) {
		if c.OnSFUStatusRecv != nil {
			c.OnSFUStatusRecv(status)
		}
	}
	return c
}

// SendQualityReport send the report, maybe lose when cannot connect
func (c *ISGLBClient) SendQualityReport(report *pb.QualityReport) {
	c.DoWithClient(func(client util.ClientStream[*pb.SyncRequest, *pb.SFUStatus]) error {
		err := client.Send(&pb.SyncRequest{Request: &pb.SyncRequest_Report{Report: report}})
		if err != nil {
			log.Errorf("QualityReport send error: %+v", err)
			return err
		}
		return nil
	})
}

// SendSFUStatus send the SFUStatus, if there is a new status should be send, the last send will be canceled
func (c *ISGLBClient) SendSFUStatus(status *pb.SFUStatus) {
	c.sendSFUStatusExec.Do(func(ctx context.Context) {
		for {
			select {
			case <-c.ctxTop.Done():
				return
			case <-ctx.Done():
				return
			default:
			}
			ok := c.DoWithClient(func(client util.ClientStream[*pb.SyncRequest, *pb.SFUStatus]) error {
				err := client.Send(&pb.SyncRequest{Request: &pb.SyncRequest_Status{Status: status}})
				if err != nil {
					log.Errorf("SFUStatus send error: %+v", err)
					return err
				}
				return nil
			})
			if ok {
				return
			}
		}
	})
}

func (c *ISGLBClient) Name() string {
	return "ISGLBClient"
}
