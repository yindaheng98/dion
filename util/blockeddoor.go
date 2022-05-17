package util

import "context"

// BlockedDoor is the door of your house, whose Lock func is a blocked function
// So the methods will MULTI-THREADED access
type BlockedDoor interface {
	// BLock Lock your Door and block until some error occurred
	BLock(param Param, OnBroken func(badGay error)) error

	// Repair same as Door.Repair
	Repair(param Param, OnBroken func(badGay error)) error

	// Update same as Door.Update
	Update(param Param, OnBroken func(badGay error)) error

	// Remove same as Door.Remove
	// !!! when Remove called, BLock should exit !!!
	Remove()
}

type unBlockedDoor struct {
	door   BlockedDoor
	ctx    context.Context
	cancel context.CancelFunc
}

func newUnBlockedDoor(door BlockedDoor) unBlockedDoor {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // routine is not running
	return unBlockedDoor{
		door:   door,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (u unBlockedDoor) Lock(param Param, OnBroken func(badGay error)) error {
	select {
	case <-u.ctx.Done(): // routine is not running
		u.ctx, u.cancel = context.WithCancel(context.Background())
		go u.routine(param, OnBroken) // run the routine
	default: // if not
	}
	return nil // just return
}

func (u unBlockedDoor) routine(param Param, OnBroken func(badGay error)) {
	for {
		select {
		case <-u.ctx.Done(): // routine should stop
			return
		default:
		}
		err := u.door.BLock(param, OnBroken)
		if err != nil {
			OnBroken(err)
		}
	}
}

func (u unBlockedDoor) Repair(param Param, OnBroken func(badGay error)) error {
	return u.door.Repair(param, OnBroken)
}

func (u unBlockedDoor) Update(param Param, OnBroken func(badGay error)) error {
	return u.door.Update(param, OnBroken)
}

func (u unBlockedDoor) Remove() {
	u.cancel()
	u.door.Remove()
}

// BlockedHouse is your house with BlockedDoor
type BlockedHouse interface {
	// NewBlockedDoor buy a new NewBlockedDoor for your House
	// the market maybe closed, if so, just return an error
	NewBlockedDoor() (BlockedDoor, error)
}

type unBlockedHouse struct {
	house BlockedHouse
}

func (u unBlockedHouse) NewDoor() (Door, error) {
	b, err := u.house.NewBlockedDoor()
	if err != nil {
		return nil, err
	}
	return unBlockedDoor{door: b}, nil
}

func NewWatchDogWithBlock(bhouse BlockedHouse) *WatchDog {
	return NewWatchDog(unBlockedHouse{house: bhouse})
}
