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

	stream     ClientStream[RequestType, ResponseType]
	cancelLast context.CancelFunc

	OnMsgRecv func(ResponseType)
}

func NewClient[RequestType, ResponseType any](factory ClientStreamFactory[RequestType, ResponseType]) *Client[RequestType, ResponseType] {
	ctx, cancal := context.WithCancel(context.Background())
	c := &Client[RequestType, ResponseType]{
		ClientStreamFactory: factory,
		ctxTop:              ctx,
		cancelTop:           cancal,

		msgReadLoopExec: NewSingleExec(),
		reconnectExec:   NewSingleWaitExec(ctx),
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
	if c.stream == nil { // no stream?
		c.reconnect() // make a stream
		return false
	}
	err := op(c.stream) // has stream? just do it
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

		if c.cancelLast != nil {
			c.cancelLast() // cancel the last stream
		}

		ctx, cancel := context.WithCancel(c.ctxTop)
		c.cancelLast = cancel
		var err error
		c.stream, err = c.NewClientStream(ctx)
		if err != nil {
			return
		}

		c.msgReadLoopExec.Do(c.msgReadLoop)

		c.connected.Store(true)
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
