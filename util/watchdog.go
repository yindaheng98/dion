package util

import (
	"context"
	"sync"
)

// House is your house
type House interface {
	// NewDoor buy a new door for your House
	NewDoor() Door
}

// Door is the door of your house
type Door interface {
	// Lock your Door when leaving your house
	// but some time, your Door can be Broken by some badGay
	// so you need a watchdog
	// Or maybe you can not Lock your Door, so you should buy a new door
	Lock(OnBroken func(badGay error)) error

	// Repair your Door after it was Broken
	// Maybe badGay is so bad that your door can not be repair
	// you can return false and your watchdog can remove this door and buy a new door for you
	Repair() bool

	// Remove your Door after it was Broken
	Remove()
}

// WatchDog is your watchdog
type WatchDog struct {
	house  House
	ctx    context.Context
	cancel context.CancelFunc

	once sync.Once
}

func NewWatchDog(house House) *WatchDog {
	ctx, cancel := context.WithCancel(context.Background())
	return &WatchDog{
		house:  house,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Watch let your dog start to watch your House
func (w *WatchDog) Watch() {
	go w.once.Do(w.watch)
}

func (w *WatchDog) watch() {
	brokenCh := make(chan error)
	var door Door = nil
	for {
		if door == nil { // do not have a door?
			door = w.house.NewDoor() // buy a new door
		}
		err := door.Lock(func(badGay error) {
			brokenCh <- badGay
		})
		if err != nil { // Can not Lock your Door
			door.Remove() // You should remove the bad door
			door = nil
			select {
			case <-w.ctx.Done(): // stop from watching your House?
				return // just exit
			default:
				continue
			}
		}
		select {
		case <-w.ctx.Done(): // stop from watching your House?
			door.Remove()
			door = nil
			return // just exit
		case <-brokenCh: // your door broken!
			if !door.Repair() { // oh no, repair it
				// badGay is so bad that your door can not be repaired
				door.Remove() // remove it
				door = nil
			}
		}
	}
}

// Leave let your dog stop from watching your House
func (w *WatchDog) Leave() {
	w.cancel()
}
