package util

import (
	"context"
	"sync"
)

type SingleWaitExec struct {
	context.Context
	ctx context.Context
	mu  sync.Mutex
}

func NewSingleWaitExec(ctx context.Context) *SingleWaitExec {
	return &SingleWaitExec{
		Context: ctx,
	}
}

// Do : do an operation.
// If nothing is running, then run it
// If something is running, then just wait it
// 很显然，如果你在 op 里面调用这个 Do 那肯定是死锁的
func (l *SingleWaitExec) Do(op func()) {
	l.mu.Lock()
	ctx := l.ctx
	if ctx == nil {
		ctx, cancel := context.WithCancel(l)
		l.ctx = ctx
		l.mu.Unlock()
		op()
		cancel()
	} else {
		select {
		case <-ctx.Done():
			ctx, cancel := context.WithCancel(l)
			l.ctx = ctx
			l.mu.Unlock()
			op()
			cancel()
		default:
			l.mu.Unlock()
			<-ctx.Done()
		}
	}
}

type SingleExec struct {
	running chan struct{}
}

func NewSingleExec() *SingleExec {
	c := make(chan struct{}, 1)
	c <- struct{}{}
	return &SingleExec{running: c}
}

// Do : do an operation.
// If nothing is running, then run it
// If something is running, then just exit
func (l *SingleExec) Do(op func()) {
	go l.do(op)
}
func (l *SingleExec) do(op func()) {
	select {
	case <-l.running:
		op()
		l.running <- struct{}{}
	default:
		return
	}
}

type SingleLatestExec struct {
	mu         sync.Mutex
	cancelLast context.CancelFunc
}

// Do : do an operation.
// If nothing is running, then run it
// If something is running, then stop the last and run the new
func (l *SingleLatestExec) Do(op func(ctx context.Context)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancelLast != nil {
		l.cancelLast()
	}
	ctx, cancel := context.WithCancel(context.Background())
	go op(ctx)
	l.cancelLast = cancel
}
