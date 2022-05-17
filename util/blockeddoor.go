package util

import "context"

// BlockedDoor is the door of your house, whose Lock func is a blocked function
// So the methods will MULTI-THREADED access
type BlockedDoor interface {
	// BLock Lock your Door and block until some error occurred
	BLock(param Param, OnBroken func(badGay error)) error

	// Update same as Door.Update
	Update(param Param, OnBroken func(badGay error)) error

	// Remove same as Door.Remove
	// !!! when Remove called, BLock should exit !!!
	Remove()
}

type unBlockedDoor struct {
	door   BlockedDoor
	param  Param
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
	u.param = param.Clone()
	select {
	case <-u.ctx.Done(): // routine is not running
		u.ctx, u.cancel = context.WithCancel(context.Background())
		go u.routine(OnBroken) // run the routine
	default: // if not
	}
	return nil // just return
}

func (u unBlockedDoor) routine(OnBroken func(badGay error)) {
	for {
		err := u.door.BLock(u.param.Clone(), OnBroken)
		select {
		case <-u.ctx.Done(): // routine should stop
			return
		default:
			if err != nil {
				OnBroken(err)
			}
		}
	}
}

func (u unBlockedDoor) Repair(param Param, OnBroken func(badGay error)) error {
	return u.Update(param, OnBroken)
}

func (u unBlockedDoor) Update(param Param, OnBroken func(badGay error)) error {
	u.param = param.Clone()
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
	return newUnBlockedDoor(b), nil
}

func NewWatchDogWithBlock(bhouse BlockedHouse) *WatchDog {
	return NewWatchDog(unBlockedHouse{house: bhouse})
}
