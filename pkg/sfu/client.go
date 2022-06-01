package sfu

import (
	"context"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"google.golang.org/grpc/metadata"
)

type ClientStreamFactory struct {
	node       *ion.Node
	peerNID    string
	parameters map[string]interface{}
	Metadata   metadata.MD
}

func (c ClientStreamFactory) NewClientStream(ctx context.Context) (util.ClientStream[*rtc.Request, *rtc.Reply], error) {
	conn, err := c.node.NewNatsRPCClient(config.ServiceSXU, c.peerNID, c.parameters)
	if err != nil {
		log.Errorf("cannot node.NewNatsRPCClient: %v", err)
		return nil, err
	}
	ctx = metadata.NewOutgoingContext(ctx, c.Metadata)
	client, err := rtc.NewRTCClient(conn).Signal(ctx)
	if err != nil {
		log.Errorf("cannot pb.NewClient: %v", err)
		return nil, err
	}
	return client, err
}

type Client struct {
	*util.Client[*rtc.Request, *rtc.Reply]
	ctxTop    context.Context
	cancelTop context.CancelFunc

	sendSFUStatusExec *util.SingleLatestExec

	client     pb.ISGLB_SyncSFUClient
	cancelLast context.CancelFunc

	OnReplyRecv func(s *rtc.Reply)
}

func NewClient(node *ion.Node, peerNID string, parameters map[string]interface{}) *Client {
	ctx, cancal := context.WithCancel(context.Background())
	c := &Client{
		Client: util.NewClient[*rtc.Request, *rtc.Reply](
			ClientStreamFactory{
				node: node, peerNID: peerNID, parameters: parameters,
			}),
		ctxTop:            ctx,
		cancelTop:         cancal,
		sendSFUStatusExec: &util.SingleLatestExec{},
	}
	c.OnMsgRecv = func(status *rtc.Reply) {
		if c.OnReplyRecv != nil {
			c.OnReplyRecv(status)
		}
	}
	return c
}

// Send send the report, maybe lose when cannot connect
func (c *Client) Send(request *rtc.Request) error {
	var err error
	c.DoWithClient(func(client util.ClientStream[*rtc.Request, *rtc.Reply]) error {
		err := client.Send(request)
		if err != nil {
			log.Errorf("rtc.Request send error: %+v", err)
			return err
		}
		return nil
	})
	return err
}

func (c *Client) Name() string {
	return "sfu.Client"
}
