package util

import (
	"context"
	"sync/atomic"
	"time"
)

type chItem[K, V comparable] struct {
	key      K
	value    V
	callback context.CancelFunc
}

type ExpireSetMap[K, V comparable] struct {
	m         map[K]map[V]*time.Timer
	expire    time.Duration
	runner    *SingleExec
	updateCh  chan chItem[K, V]
	deleteCh  chan chItem[K, V]
	onDeleted atomic.Value
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewExpireSetMap[K, V comparable](expire time.Duration) *ExpireSetMap[K, V] {
	ctx, cancel := context.WithCancel(context.Background())
	return &ExpireSetMap[K, V]{
		m:        make(map[K]map[V]*time.Timer),
		expire:   expire,
		runner:   NewSingleExec(),
		deleteCh: make(chan chItem[K, V], 64),
		updateCh: make(chan chItem[K, V], 64),
		ctx:      ctx, cancel: cancel,
	}
}

func (m *ExpireSetMap[K, V]) Start() {
	m.runner.Do(func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case i := <-m.updateCh:
				key, value, callback := i.key, i.value, i.callback
				m.handleUpdate(m.ctx, key, value, callback)
			case i := <-m.deleteCh:
				key, value, callback := i.key, i.value, i.callback
				m.handleDelete(key, value, callback)
			}
		}
	})
}

func (m *ExpireSetMap[K, V]) Stop() {
	m.cancel()
}

func (m *ExpireSetMap[K, V]) handleUpdate(ctx context.Context, key K, value V, callback func()) {
	if set, ok := m.m[key]; ok { // set exist?
		if timer, ok := set[value]; ok { // timer exist?
			timer.Reset(m.expire) // just reset it
		} else { // timer not exist?
			timer, start := m.newTimer(ctx, key, value) // create it
			set[value] = timer
			start()
		}
	} else { // set not exist?
		timer, start := m.newTimer(ctx, key, value)
		m.m[key] = map[V]*time.Timer{value: timer} // create the set
		start()
	}
	callback()
}

func (m *ExpireSetMap[K, V]) newTimer(ctx context.Context, key K, value V) (timer *time.Timer, start func()) {
	timer = time.NewTimer(m.expire)
	start = func() {
		go func(ctx context.Context, timer *time.Timer, key K, value V) {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				m.deleteCh <- chItem[K, V]{
					key:      key,
					value:    value,
					callback: func() {},
				}
				return
			}
		}(ctx, timer, key, value)
	}
	return
}

func (m *ExpireSetMap[K, V]) handleDelete(key K, value V, callback func()) {
	if set, ok := m.m[key]; ok {
		delete(set, value)
		if len(set) <= 0 {
			delete(m.m, key)
		}
		if handler := m.onDeleted.Load().(func(K, V)); handler != nil {
			handler(key, value)
		}
	}
	callback()
}

// Update a item and wait for it done
func (m *ExpireSetMap[K, V]) Update(key K, value V) {
	ctx, cancel := context.WithCancel(context.Background())
	m.updateCh <- chItem[K, V]{
		key:      key,
		value:    value,
		callback: cancel,
	}
	<-ctx.Done()
}

func (m *ExpireSetMap[K, V]) OnDelete(handler func(K, V)) {
	m.onDeleted.Store(handler)
}
