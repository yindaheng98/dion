package sfu

import (
	"context"
	"fmt"
	"github.com/yindaheng98/dion/algorithms/impl/random"
	pb "github.com/yindaheng98/dion/proto"
	"github.com/yindaheng98/dion/util"
	"math/rand"
	"sync"
	"testing"
)

type testForwardTrackRoutineFactory struct {
	n       uint32
	running map[<-chan util.ForwardTrackItem]uint32
	sync.Mutex
}

func (t *testForwardTrackRoutineFactory) ForwardTrackRoutine(Ctx context.Context, updateCh <-chan util.ForwardTrackItem) {
	var item util.ForwardTrackItem
	t.Lock()
	n := t.n
	t.n += 1
	if _, ok := t.running[updateCh]; !ok {
		t.running[updateCh] = 0
	}
	t.running[updateCh] += 1
	r := t.running[updateCh]
	fmt.Printf("No.%d started %d times\n", n, r)
	t.Unlock()
	first := true
	for {
		select {
		case <-Ctx.Done(): // this forwarding should exit
			fmt.Printf("No.%d exited\n", n)
			return
		case item = <-updateCh: // get item from update channel or retry channel
			if first {
				fmt.Printf("No.%d starting %+v\n", n, item)
				first = false
			} else {
				fmt.Printf("No.%d updating %+v\n", n, item)
			}
		}
	}
}

func TestForwardController(t *testing.T) {
	f := NewForwardController(&testForwardTrackRoutineFactory{running: map[<-chan util.ForwardTrackItem]uint32{}})
	var ts = []*pb.ForwardTrack{random.RandForwardTrack()}
	for i := 0; i < 100; i++ {
		switch rand.Intn(3) {
		case 0:
			k := random.RandForwardTrack()
			t.Logf("f.StartForwardTrack(%+v)", k)
			f.StartForwardTrack(k)
			ts = append(ts, k)
		case 1:
			oldT := random.RandForwardTrack()
			newT := random.RandForwardTrack()
			if random.RandBool() {
				oldT = ts[rand.Intn(len(ts))]
			}
			if random.RandBool() {
				newT = ts[rand.Intn(len(ts))]
			}
			t.Logf("f.ReplaceForwardTrack(%+v, %+v)", oldT, newT)
			f.ReplaceForwardTrack(oldT, newT)
		case 2:
			if random.RandBool() {
				k := random.RandForwardTrack()
				t.Logf("f.StopForwardTrack(%+v)", k)
				f.StopForwardTrack(random.RandForwardTrack())
			} else {
				k := ts[rand.Intn(len(ts))]
				t.Logf("f.StopForwardTrack(%+v)", k)
				f.StopForwardTrack(k)
			}
		}
	}
}
