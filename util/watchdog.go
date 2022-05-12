package util

type Door interface {
}

type WatchDog struct {
}

// Watch let your dog start to watch your Door
func (w *WatchDog) Watch() {

}

// Leave let your dog stop from watching your Door
func (w *WatchDog) Leave() {

}
