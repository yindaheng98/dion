package isglb

import (
	"context"
	log "github.com/pion/ion-log"
	sdk "github.com/pion/ion-sdk-go"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"google.golang.org/grpc/metadata"
	"sync/atomic"
)

type ISGLBClient struct {
	sdk.Service
	node       *ion.Node
	peerNID    string
	parameters map[string]interface{}
	ctxTop     context.Context
	cancelTop  context.CancelFunc

	connected         atomic.Value
	msgReadLoopExec   *util.SingleExec
	sendSFUStatusExec *util.SingleLatestExec
	reconnectExec     *util.SingleWaitExec

	client     pb.ISGLB_SyncSFUClient
	cancelLast context.CancelFunc

	Metadata metadata.MD

	OnSFUStatusRecv func(s *pb.SFUStatus)
}

func NewISGLBClient(node *ion.Node, peerNID string, parameters map[string]interface{}) *ISGLBClient {
	ctx, cancal := context.WithCancel(context.Background())
	c := &ISGLBClient{
		node:              node,
		peerNID:           peerNID,
		parameters:        parameters,
		ctxTop:            ctx,
		cancelTop:         cancal,
		msgReadLoopExec:   util.NewSingleExec(),
		sendSFUStatusExec: &util.SingleLatestExec{},
		reconnectExec:     util.NewSingleWaitExec(ctx),
	}
	c.connected.Store(false)
	return c
}

// doWithStream do something with c.stream
func (c *ISGLBClient) doWithClient(op func(client pb.ISGLB_SyncSFUClient) error) bool {
	select {
	case <-c.ctxTop.Done(): // should exit now?
		return true // exit
	default:
	}
	if c.client == nil { // no client?
		c.reconnect() // make a client
		return false
	}
	err := op(c.client) // has client? just do it
	if err != nil {     // error? should reconnect
		select {
		case <-c.ctxTop.Done(): // should exit now?
			return true // do not reconnect
		default:
		}
		c.reconnect() // reconnect
		return false
	}
	return true
}

func (c *ISGLBClient) msgReadLoop() {
	for {
		select {
		case <-c.ctxTop.Done(): // should exit now?
			return // exit
		default:
		}
		// receive msg
		c.doWithClient(func(client pb.ISGLB_SyncSFUClient) error {
			s, err := client.Recv() // Receive a SyncRequest
			if err != nil {
				log.Errorf("SyncRequest receive error %+v", err)
				return err
			}
			c.OnSFUStatusRecv(s)
			return nil
		})
	}
}

func (c *ISGLBClient) reconnect() {
	// c.Do: 如果当前正在重连，就等待重连完成；如果当前不在重连，就开始重连直到完成
	c.reconnectExec.Do(func() {
		c.connected.Store(false)
		log.Infof("ISGLBClient connecting......")

		if c.cancelLast != nil {
			c.cancelLast() // cancel the last stream
		}

		conn, err := c.node.NewNatsRPCClient(config.ServiceISGLB, c.peerNID, c.parameters)
		if err != nil {
			log.Errorf("cannot NewNatsRPCClient: %v", err)
			return
		}

		ctx, cancel := context.WithCancel(c.ctxTop)
		c.cancelLast = cancel

		ctx = metadata.NewOutgoingContext(ctx, c.Metadata)
		c.client, err = pb.NewISGLBClient(conn).SyncSFU(ctx)
		if err != nil {
			log.Errorf("cannot NewISGLBClient: %v", err)
			return
		}

		c.msgReadLoopExec.Do(c.msgReadLoop)

		c.connected.Store(true)
		log.Infof("ISGLBClient connected!")
	})
}

// SendQualityReport send the report, maybe lose when cannot connect
func (c *ISGLBClient) SendQualityReport(report *pb.QualityReport) {
	c.doWithClient(func(client pb.ISGLB_SyncSFUClient) error {
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
			ok := c.doWithClient(func(client pb.ISGLB_SyncSFUClient) error {
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

func (c *ISGLBClient) SendSyncRequest(r *pb.SyncRequest) {
	select {
	case <-c.ctxTop.Done():
		return
	default:
	}
	if c.client == nil {
		c.reconnect()
	} else {
		err := c.client.Send(r)
		if err != nil {
			c.reconnect()
		}
	}
}

func (c *ISGLBClient) Close() {
	c.cancelTop()
}

func (c *ISGLBClient) Name() string {
	return "ISGLBClient"
}

func (c *ISGLBClient) Connect() {
	c.reconnect()
}

func (c *ISGLBClient) Connected() bool {
	return c.connected.Load().(bool)
}
