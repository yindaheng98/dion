package util

import "context"

// BlockedDoor is the door of your house, whose Lock func is a blocked function
// So the methods will MULTI-THREADED access
type BlockedDoor interface {
	// BLock Lock your Door and block until some error occurred
	// 如果出错了会直接再次调用BLock，而不是新建BlockedDoor
	BLock(init Param) error

	// Update same as Door.Update, but you should not return error
	// what if Update not success? you should return the error from BLock
	Update(param Param)

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

// Lock will never return error
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
		err := u.door.BLock(u.param.Clone())
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

// Repair will never return error
func (u unBlockedDoor) Repair(param Param) error {
	return u.Update(param)
}

// Update will never return error
func (u unBlockedDoor) Update(param Param) error {
	u.param = param.Clone()
	u.door.Update(param)
	return nil
}

// Remove will only called in WatchDog.Leave
func (u unBlockedDoor) Remove() {
	u.cancel()
	u.door.Remove()
}

type unBlockedHouse struct {
	door BlockedDoor
}

func (u unBlockedHouse) NewDoor() (Door, error) {
	return newUnBlockedDoor(u.door), nil
}

func NewWatchDogWithBlock(door BlockedDoor) *WatchDog {
	return NewWatchDog(unBlockedHouse{door: door})
}
