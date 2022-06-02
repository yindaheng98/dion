package util

import (
	"context"
	"sync/atomic"
)

type ClientStreamFactory[RequestType, ResponseType any] interface {
	NewClientStream(ctx context.Context) (ClientStream[RequestType, ResponseType], error)
}

type ClientStream[RequestType, ResponseType any] interface {
	Send(RequestType) error
	Recv() (ResponseType, error)
}

type Client[RequestType, ResponseType any] struct {
	ClientStreamFactory[RequestType, ResponseType]
	ctxTop    context.Context
	cancelTop context.CancelFunc

	connected       atomic.Value
	msgReadLoopExec *SingleExec
	reconnectExec   *SingleWaitExec

	stream         atomic.Value
	cancelLast     context.CancelFunc
	reconnectTimes uint32

	OnMsgRecv   func(ResponseType)
	OnReconnect func()
}

func NewClient[RequestType, ResponseType any](factory ClientStreamFactory[RequestType, ResponseType]) *Client[RequestType, ResponseType] {
	ctx, cancal := context.WithCancel(context.Background())
	c := &Client[RequestType, ResponseType]{
		ClientStreamFactory: factory,
		ctxTop:              ctx,
		cancelTop:           cancal,

		msgReadLoopExec: NewSingleExec(),
		reconnectExec:   NewSingleWaitExec(ctx),

		reconnectTimes: 0,
	}
	c.connected.Store(false)
	return c
}

// DoWithClient do something with c.stream
func (c *Client[RequestType, ResponseType]) DoWithClient(op func(client ClientStream[RequestType, ResponseType]) error) bool {
	select {
	case <-c.ctxTop.Done(): // should exit now?
		return true // exit
	default:
	}
	stream := c.stream.Load()
	if stream == nil { // no stream?
		c.reconnect() // make a stream
		return false
	}
	err := op(stream.(ClientStream[RequestType, ResponseType])) // has stream? just do it
	if err != nil {                                             // error? should reconnect
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

func (c *Client[RequestType, ResponseType]) msgReadLoop() {
	for {
		select {
		case <-c.ctxTop.Done(): // should exit now?
			return // exit
		default:
		}
		// receive msg
		c.DoWithClient(func(client ClientStream[RequestType, ResponseType]) error {
			s, err := client.Recv() // Receive a Request
			if err != nil {
				return err
			}
			c.OnMsgRecv(s)
			return nil
		})
	}
}

func (c *Client[RequestType, ResponseType]) reconnect() {
	// c.Do: 如果当前正在重连，就等待重连完成；如果当前不在重连，就开始重连直到完成
	c.reconnectExec.Do(func() {
		c.connected.Store(false)
		atomic.AddUint32(&c.reconnectTimes, 1)

		if c.cancelLast != nil {
			c.cancelLast() // cancel the last stream
		}

		ctx, cancel := context.WithCancel(c.ctxTop)
		c.cancelLast = cancel
		var err error
		stream, err := c.NewClientStream(ctx)
		if err != nil {
			return
		}
		c.stream.Store(stream)

		c.msgReadLoopExec.Do(c.msgReadLoop)

		c.connected.Store(true)
		if c.OnReconnect != nil {
			c.OnReconnect()
		}
	})
}

// Send send the request, maybe lose when cannot connect
func (c *Client[RequestType, ResponseType]) Send(request RequestType) {
	c.DoWithClient(func(client ClientStream[RequestType, ResponseType]) error {
		err := client.Send(request)
		if err != nil {
			return err
		}
		return nil
	})
}

func (c *Client[RequestType, ResponseType]) Close() {
	c.cancelTop()
}

func (c *Client[RequestType, ResponseType]) Connect() {
	c.msgReadLoopExec.Do(c.msgReadLoop)
}

func (c *Client[RequestType, ResponseType]) Connected() bool {
	return c.connected.Load().(bool)
}

func (c *Client[RequestType, ResponseType]) Reconnect() {
	before := atomic.LoadUint32(&c.reconnectTimes) // reconnect times before reconnect
	c.reconnect()
	after := atomic.LoadUint32(&c.reconnectTimes) // reconnect times after reconnect
	// if reconnect times not update, it means c.reconnect() just wait for another c.reconnect()
	if before >= after {
		c.reconnect() // should reconnect again
	}
}
