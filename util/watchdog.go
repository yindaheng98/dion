package util

import (
	"context"
	"sync"
)

type Param interface {
	Clone() Param
}

// House is your house
type House interface {
	// NewDoor buy a new door for your House
	// the market maybe closed, if so, just return an error
	NewDoor() (Door, error)
}

// Door is the door of your house
type Door interface {
	// Lock your Door when leaving your house
	// but some time, your Door can be Broken by some badGay
	// so you need a watchdog
	// Or maybe you can not Lock your Door, so you should buy a new door
	Lock(param Param, OnBroken func(badGay error)) error

	// Repair your Door after it was Broken
	// Maybe badGay is so bad that your door can not be repair
	// you can return false and your watchdog can remove this door and buy a new door for you
	Repair(param Param, OnBroken func(badGay error)) error

	// Update your Door even if it was not Broken while watching
	Update(param Param, OnBroken func(badGay error)) error

	// Remove your Door after it was Broken
	Remove()
}

// WatchDog is your watchdog
type WatchDog struct {
	house  House
	ctx    context.Context
	cancel context.CancelFunc

	once sync.Once

	param    Param
	updateCh chan Param
}

func NewWatchDog(house House) *WatchDog {
	ctx, cancel := context.WithCancel(context.Background())
	return &WatchDog{
		house:    house,
		ctx:      ctx,
		cancel:   cancel,
		updateCh: make(chan Param, 1),
	}
}

// Watch let your dog start to watch your House
func (w *WatchDog) Watch(init Param) {
	w.param = init.Clone()
	go w.once.Do(w.watch)
}

func (w *WatchDog) watch() {
	brokenCh := make(chan error, 1)
	OnBroken := func(badGay error) {
		select {
		case brokenCh <- badGay:
		default:
		}
	}
	var door Door = nil
	for {
		if door == nil { // do not have a door?
			var err error
			door, err = w.house.NewDoor() // buy a new door
			if err != nil {               // market closed?
				if door != nil {
					door.Remove() // your door is fake, do not use it!
					door = nil
				}
				select {
				case <-w.ctx.Done(): // stop from watching your House?
					return // just exit
				default:
					continue
				}
			}
		}
		err := door.Lock(w.param.Clone(), OnBroken)
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
			if err := door.Repair(w.param.Clone(), OnBroken); err != nil { // oh no, repair it
				// badGay is so bad that your door can not be repaired
				door.Remove() // remove it
				door = nil
			}
		case param := <-w.updateCh:
			w.param = param
			if err := door.Update(w.param.Clone(), OnBroken); err != nil { // update it
				// Cannot?
				door.Remove() // remove it
				door = nil
			}
		}
	}
}

func (w *WatchDog) Update(param Param) {
	select {
	case w.updateCh <- param.Clone():
	default:
		select {
		case <-w.updateCh:
		default:
		}
		select {
		case w.updateCh <- param.Clone():
		default:
		}
	}
}

// Leave let your dog stop from watching your House
func (w *WatchDog) Leave() {
	w.cancel()
}
