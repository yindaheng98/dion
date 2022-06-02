package sfu

import (
	"context"
	"errors"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/rtc"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"google.golang.org/grpc/metadata"
	"sync/atomic"
)

type param struct {
	peerNID    string
	parameters map[string]interface{}
}

type ClientStreamFactory struct {
	node     *ion.Node
	param    atomic.Value
	Metadata metadata.MD
}

func NewClientStreamFactory(node *ion.Node) *ClientStreamFactory {
	c := &ClientStreamFactory{
		node: node,
	}
	c.param.Store(param{
		peerNID:    "*",
		parameters: map[string]interface{}{},
	})
	return c
}

func (c *ClientStreamFactory) NewClientStream(ctx context.Context) (util.ClientStream[*rtc.Request, *rtc.Reply], error) {
	p := c.param.Load().(param)
	peerNID, parameters := p.peerNID, p.parameters
	conn, err := c.node.NewNatsRPCClient(config.ServiceSXU, peerNID, parameters)
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

func (c *ClientStreamFactory) Switch(peerNID string, parameters map[string]interface{}) {
	c.param.Store(param{
		peerNID:    peerNID,
		parameters: parameters,
	})
}

type Client struct {
	*util.Client[*rtc.Request, *rtc.Reply]
	ctxTop    context.Context
	cancelTop context.CancelFunc

	sendSFUStatusExec *util.SingleLatestExec

	client     pb.ISGLB_SyncSFUClient
	cancelLast context.CancelFunc
}

func NewClient(node *ion.Node) *Client {
	ctx, cancal := context.WithCancel(context.Background())
	c := &Client{
		Client:            util.NewClient[*rtc.Request, *rtc.Reply](NewClientStreamFactory(node)),
		ctxTop:            ctx,
		cancelTop:         cancal,
		sendSFUStatusExec: &util.SingleLatestExec{},
	}
	return c
}

// Send send the report, maybe lose when cannot connect
func (c *Client) Send(request *rtc.Request) error {
	var err error
	exec := false
	c.DoWithClient(func(client util.ClientStream[*rtc.Request, *rtc.Reply]) error {
		exec = true
		err = client.Send(request)
		if err != nil {
			log.Errorf("rtc.Request send error: %+v", err)
			return err
		}
		return nil
	})
	if !exec {
		return errors.New("rtc.Request not send")
	} else {
		return err
	}
}

func (c *Client) Name() string {
	return "sfu.Client"
}

func (c *Client) Switch(peerNID string, parameters map[string]interface{}) {
	c.Client.ClientStreamFactory.(*ClientStreamFactory).Switch(peerNID, parameters)
	c.Reconnect()
}
