package util

import (
	"context"
	"sync"
)

type SingleWaitExec struct {
	running bool
	sync.WaitGroup
	mu sync.Mutex
}

// Do : do an operation.
// If nothing is running, then run it
// If something is running, then just wait it
func (l *SingleWaitExec) Do(op func()) {
	l.mu.Lock()
	if !l.running { // nothing is running, I should run it
		l.running = true
		l.Add(1)
		l.mu.Unlock()
		op()
		l.Done()
		l.mu.Lock()
		l.running = false
		l.mu.Unlock()
	} else { // means something is doing, I should wait it
		l.mu.Unlock()
		l.Wait()
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
