package util

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/yindaheng98/dion/algorithms/impl/random"
	pb "github.com/yindaheng98/dion/proto"
)

func randTracks(n int) []*pb.ProceedTrack {
	ts := make([]*pb.ProceedTrack, n)
	for i := 0; i < n; i++ {
		ts[i] = random.RandProceedTrack()
	}
	return ts
}

const N = 100

func TestDisorderSet(t *testing.T) {
	set := NewDisorderSet()
	templates := randTracks(N)
	for i := 0; i < N; i++ {
		set.Add(ProceedTrackItem{
			Track: templates[i],
		})
	}
	fmt.Printf("set: %+v\n", ItemList(set.Sort()).ToProceedTracks())
	for i := 0; i < N/2; i++ {
		set.Del(ProceedTrackItem{
			Track: templates[rand.Int31n(N)],
		})
	}
	fmt.Printf("set: %+v\n", ItemList(set.Sort()).ToProceedTracks())

	set2 := NewDisorderSet()
	set2.Del(ProceedTrackItem{
		Track: templates[rand.Int31n(N)],
	})
	for i := 0; i < N/2; i++ {
		set2.Add(ProceedTrackItem{
			Track: templates[rand.Int31n(N)],
		})
	}
	fmt.Printf("set2: %+v\n", ItemList(set2.Sort()).ToProceedTracks())
	diff := make(ItemList, N/2)
	for i := 0; i < N/2; i++ {
		diff[i] = ProceedTrackItem{
			Track: templates[rand.Int31n(N)],
		}
		random.RandChangeProceedTrack(diff[i].(ProceedTrackItem).Track)
	}
	fmt.Printf("diff: %+v\n", ItemList(diff).ToProceedTracks())
	add, del, replace := set2.Update(diff)
	fmt.Printf("Add: %+v\n", ItemList(add).ToProceedTracks())
	fmt.Printf("Del: %+v\n", ItemList(del).ToProceedTracks())
	for _, r := range replace {
		fmt.Printf("%+v -> %+v\n", r.Old.(ProceedTrackItem).Track, r.New.(ProceedTrackItem).Track)
	}
}
