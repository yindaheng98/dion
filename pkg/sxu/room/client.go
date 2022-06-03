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
	"google.golang.org/protobuf/proto"
	"sync/atomic"
	"time"
)

type Client struct {
	sdk.Service
	factory   ClientFactory
	ctxTop    context.Context
	cancelTop context.CancelFunc

	session         atomic.Value
	connected       atomic.Value
	keepAliveExec   *util.SingleExec
	reconnectExec   *util.SingleWaitExec
	manualReconnect chan struct{}

	conn *rpc.Client
}

func NewClient(factory ClientFactory) *Client {
	ctx, cancal := context.WithCancel(context.Background())
	c := &Client{
		factory:   factory,
		ctxTop:    ctx,
		cancelTop: cancal,

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
		default:
			t := time.After(config.ClientSessionLifeCycle)
			ctx, cancel := context.WithCancel(context.Background())
			go func(cancel context.CancelFunc) {
				// receive msg
				c.doWithClient(func(client pb.RoomClient) error {
					defer cancel()
					if session := c.session.Load(); session != nil {
						reply, err := client.ClientHealth(c.ctxTop, session.(*pb.ClientNeededSession)) // keep alive
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
			}(cancel)
			select {
			case <-ctx.Done():
				<-t
			case <-t:
				log.Errorf("ClientHealth time out")
				c.reconnect()
			}
		}
	}
}

func (c *Client) reconnect() {
	// c.Do: 如果当前正在重连，就等待重连完成；如果当前不在重连，就开始重连直到完成
	c.reconnectExec.Do(func() {
		log.Infof("room.Client connecting......")
		if c.conn != nil {
			_ = c.conn.Close()
			// TODO: nats-grpc的问题导致这里偶尔出现fatal error: concurrent map iteration and map write，已给原作者提pull request
		}

		var conn = c.factory.NewClient() // select a node

		if conn == nil {
			log.Errorf("no client can be use")
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
	c.keepAliveExec.Do(c.keepAlive)
}

func (c *Client) Connected() bool {
	return c.connected.Load().(bool)
}
