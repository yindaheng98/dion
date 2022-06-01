package room

import (
	"context"
	"fmt"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	log "github.com/pion/ion-log"
	sdk "github.com/pion/ion-sdk-go"
	"github.com/yindaheng98/dion/config"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"github.com/yindaheng98/dion/util/ion"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"sync/atomic"
	"time"
)

type Client struct {
	sdk.Service
	selector   Selector
	node       *ion.Node
	parameters map[string]interface{}
	ctxTop     context.Context
	cancelTop  context.CancelFunc

	session         atomic.Value[*pb.ClientNeededSession]
	connected       atomic.Value[bool]
	keepAliveExec   *util.SingleExec
	reconnectExec   *util.SingleWaitExec
	manualReconnect chan struct{}

	conn grpc.ClientConnInterface
}

func NewClient(node *ion.Node, selector Selector, parameters map[string]interface{}) *Client {
	ctx, cancal := context.WithCancel(context.Background())
	c := &Client{
		selector:   selector,
		node:       node,
		parameters: parameters,
		ctxTop:     ctx,
		cancelTop:  cancal,

		keepAliveExec:   util.NewSingleExec(),
		reconnectExec:   util.NewSingleWaitExec(ctx),
		manualReconnect: make(chan struct{}, 1),
	}
	c.connected.Store(false)
	return c
}

// doWithStream do something with c.conn
func (c *Client) doWithClient(op func(client pb.RoomClient) error) {
	select {
	case <-c.ctxTop.Done(): // should exit now?
		return // exit
	default:
	}
	if c.conn == nil { // no client?
		c.reconnect() // make a client
		return
	}
	err := op(pb.NewRoomClient(c.conn)) // has client? just do it
	if err != nil {                     // error? should reconnect
		select {
		case <-c.ctxTop.Done(): // should exit now?
			return // do not reconnect
		default:
		}
		c.reconnect() // reconnect
		return
	}
	return
}

func (c *Client) keepAlive() {
	for {
		select {
		case <-c.ctxTop.Done(): // should exit now?
			return // exit
		case <-c.manualReconnect: // want refresh connection?
			c.reconnect()
		case <-time.After(config.ClientSessionLifeCycle):
		}
		// receive msg
		c.doWithClient(func(client pb.RoomClient) error {
			if session := c.session.Load().(*pb.ClientNeededSession); session != nil {
				reply, err := client.ClientHealth(c.ctxTop, session) // keep alive
				if err != nil {
					log.Errorf("ClientHealth error %+v", err)
					return err
				}
				if !reply.Ok {
					log.Errorf("ClientHealth return false")
					return fmt.Errorf("ClientHealth return false")
				}
			}
			return nil
		})
	}
}

func (c *Client) reconnect() {
	// c.Do: 如果当前正在重连，就等待重连完成；如果当前不在重连，就开始重连直到完成
	c.reconnectExec.Do(func() {
		log.Infof("room.Client connecting......")

		nodes := c.node.GetNeighborNodes()
		if len(nodes) <= 0 {
			log.Errorf("there is no nodes can be connect")
			return
		}
		nodel := c.selector.Select(nodes) // select a node
		if len(nodes) <= 0 {
			log.Errorf("there is no nodes to be connect")
			return
		}

		var err error
		var conn *rpc.Client
		for _, node := range nodel {
			conn, err = c.node.NewNatsRPCClient(config.ServiceSXU, node.NID, c.parameters)
			if err != nil {
				log.Errorf("cannot NewNatsRPCClient: %v, try next", err)
			} else {
				break
			}
		}

		if conn == nil {
			log.Errorf("all NewNatsRPCClient attemp failed")
			return
		}

		c.conn = conn
		c.keepAliveExec.Do(c.keepAlive)
		c.connected.Store(true)
		log.Infof("room.Client connected!")
	})
}

// UpdateSession update the needed session
func (c *Client) UpdateSession(session *pb.ClientNeededSession) {
	c.session.Store(proto.Clone(session))
}

func (c *Client) RefreshConn() {
	select {
	case c.manualReconnect <- struct{}{}:
	default:
	}
}

func (c *Client) Close() {
	c.cancelTop()
}

func (c *Client) Name() string {
	return "room.Client"
}

func (c *Client) Connect() {
	c.reconnect()
}

func (c *Client) Connected() bool {
	return c.connected.Load().(bool)
}
