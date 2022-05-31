package util

import (
	"context"
	"sync"
)

// WatchDog is your watchdog
type WatchDog[P Param] interface {

	// Watch let your dog start to watch your House
	Watch(init P)

	// Leave let your dog stop from watching your House
	Leave()

	Update(param P)
}

type Param interface {
	Clone() Param
}

// House is your house
type House[P Param] interface {
	// NewDoor buy a new door for your House
	// the market maybe closed, if so, just return an error
	NewDoor() (UnblockedDoor[P], error)
}

// UnblockedDoor is the door of your house
// All the methods will SINGLE-THREADED access
type UnblockedDoor[P Param] interface {
	// Lock your Door when leaving your house
	// but some time, your Door can be Broken by some badGay
	// so you need a watchdog
	// Or maybe you can not Lock your Door, so you should buy a new door
	Lock(param P, OnBroken func(badGay error)) error

	// Repair your Door after it was Broken
	// Maybe badGay is so bad that your door can not be repair
	// you can return false and your watchdog can remove this door and buy a new door for you
	Repair(param P) error

	// Update your Door even if it was not Broken while watching
	Update(param P) error

	// Remove your Door after it was Broken
	Remove()
}

type WatchDogWithUnblockedDoor[P Param] struct {
	house  House[P]
	ctx    context.Context
	cancel context.CancelFunc

	once sync.Once

	param    P
	updateCh chan P
}

func NewWatchDogWithUnblockedDoor[P Param](house House[P]) WatchDog[P] {
	ctx, cancel := context.WithCancel(context.Background())
	return &WatchDogWithUnblockedDoor[P]{
		house:    house,
		ctx:      ctx,
		cancel:   cancel,
		updateCh: make(chan P, 1),
	}
}

func (w *WatchDogWithUnblockedDoor[P]) Watch(init P) {
	w.param = init.Clone().(P)
	go w.once.Do(w.watch)
}

func (w *WatchDogWithUnblockedDoor[P]) watch() {
	brokenCh := make(chan error, 1)
	OnBroken := func(badGay error) {
		select {
		case brokenCh <- badGay:
		default:
		}
	}
	var door UnblockedDoor[P] = nil
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
		err := door.Lock(w.param.Clone().(P), OnBroken)
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
			if err := door.Repair(w.param.Clone().(P)); err != nil { // oh no, repair it
				// badGay is so bad that your door can not be repaired
				door.Remove() // remove it
				door = nil
			}
		case param := <-w.updateCh:
			w.param = param
			if err := door.Update(w.param.Clone().(P)); err != nil { // update it
				// Cannot?
				door.Remove() // remove it
				door = nil
			}
		}
	}
}

func (w *WatchDogWithUnblockedDoor[P]) Update(param P) {
	select {
	case w.updateCh <- param.Clone().(P):
	default:
		select {
		case <-w.updateCh:
		default:
		}
		select {
		case w.updateCh <- param.Clone().(P):
		default:
		}
	}
}

func (w *WatchDogWithUnblockedDoor[P]) Leave() {
	w.cancel()
}
