package util

import "context"

// BlockedHouse is your house
type BlockedHouse interface {
	// NewDoor buy a new door for your House
	// the market maybe closed, if so, just return an error
	NewDoor() (BlockedDoor, error)
}

// BlockedDoor is the door of your house, whose Lock func is a blocked function
// So the methods will MULTI-THREADED access
type BlockedDoor interface {
	// BLock Lock your Door and block until some error occurred
	// 错！→×××如果出错了会直接再次调用BLock，而不是新建BlockedDoor×××
	// 多次调用了BLock，Update如何与之同步？×应该BLock只调用一次
	// 所以每个BlockedDoor中的BLock会且仅会调用一次
	// 所以请放心地把临时变量放在BlockedDoor中
	BLock(init Param) error

	// Update same as UnblockedDoor.Update
	// will retry until success if return error
	Update(param Param) error

	// Remove same as UnblockedDoor.Remove
	// !!! when Remove called, BLock should exit !!!
	Remove()
}

// BlockedDoor的创建（即NewDoor）和Update必须同步于一个线程中

type WatchDogWithBlockedDoor struct {
	house    BlockedHouse
	updateCh chan Param

	ctx    context.Context
	cancel context.CancelFunc
}

func NewWatchDogWithBlockedDoor(house BlockedHouse) WatchDog {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // routine is not running
	return WatchDogWithBlockedDoor{
		house:    house,
		updateCh: make(chan Param, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (u WatchDogWithBlockedDoor) Watch(param Param) {
	select {
	case <-u.ctx.Done(): // routine is not running
		u.ctx, u.cancel = context.WithCancel(context.Background())
		go routine(u.house, param.Clone(), u.ctx, u.updateCh) // run the routine
	default: // routine is running, do nothing
	}
}

func (u WatchDogWithBlockedDoor) Update(param Param) {
	select {
	case u.updateCh <- param.Clone():
	default:
		select {
		case <-u.updateCh:
		default:
		}
		select {
		case u.updateCh <- param.Clone():
		default:
		}
	}
}

func (u WatchDogWithBlockedDoor) Leave() {
	u.cancel()
}

func routine(house BlockedHouse, init Param, ctx context.Context, updateCh <-chan Param) {
	param := init.Clone()
L:
	for {
		select {
		case <-ctx.Done(): // should stop?
			return // just exit
		default:
		} // if not

		// init the door
		door, err := house.NewDoor()
		if err != nil { // error?
			select {
			case <-ctx.Done(): // should stop?
				return // exit
			default:
			} // if not
			continue // retry until success
		}

		// start the door
		errCh := make(chan error, 1)
		go func(errCh chan<- error, param Param) {
			errCh <- door.BLock(param)
		}(errCh, param.Clone())

		// wait for the exit or update
		retryCh := make(chan Param, 1)
		retryPush := func(param Param) { // 无阻塞进retryCh
			select {
			case retryCh <- param:
			default:
				select {
				case <-retryCh:
				default:
				}
				select {
				case retryCh <- param:
				default:
				}
			}
		}
		for {
			select {
			case <-errCh: // error? door.BLock should have exited
				continue L // restart it
			case <-ctx.Done(): // routine should stop?
				door.Remove() // stop it
				return
			case param = <-updateCh: // updateCh里有东西？
				retryPush(param) // updateCh里的东西直接进retryCh
			case param = <-retryCh: // retryCh里有东西？
				select {
				case param = <-updateCh: // 先确认updateCh里没东西
					retryPush(param) // 有东西就进retryCh
				default: // 确认updateCh里没东西了就继续操作
					if err := door.Update(param); err != nil { // error?
						retryPush(param) // 出错就进retryCh
					}
				}
			}
		}
	}
}
