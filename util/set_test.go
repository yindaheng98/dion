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

const N = 10

func TestDisorderSet(t *testing.T) {
	set := NewDisorderSet[ProceedTrackItem]()
	templates := randTracks(N)
	for i := 0; i < N; i++ {
		set.Add(ProceedTrackItem{
			Track: templates[i],
		})
	}
	for _, i := range ProceedTrackItemList(set.Sort()).ToProceedTracks() {
		fmt.Printf("after add, set: %+v\n", i)
	}
	fmt.Println("")
	for i := 0; i < N/2; i++ {
		set.Del(ProceedTrackItem{
			Track: templates[rand.Int31n(N)],
		})
	}
	for _, i := range ProceedTrackItemList(set.Sort()).ToProceedTracks() {
		fmt.Printf("after del, set: %+v\n", i)
	}
	fmt.Println("")

	set2 := NewDisorderSet[ProceedTrackItem]()
	set2.Del(ProceedTrackItem{
		Track: templates[rand.Int31n(N)],
	})
	for i := 0; i < N/2; i++ {
		set2.Add(ProceedTrackItem{
			Track: templates[rand.Int31n(N)],
		})
	}
	for _, i := range ProceedTrackItemList(set2.Sort()).ToProceedTracks() {
		fmt.Printf("set2: %+v\n", i)
	}
	fmt.Println("")

	diff := make(ProceedTrackItemList, N/2)
	for i := 0; i < N/2; i++ {
		diff[i] = ProceedTrackItem{
			Track: templates[rand.Int31n(N)],
		}
		random.RandChangeProceedTrack(diff[i].Track)
	}
	for _, i := range diff {
		fmt.Printf("diff: %+v\n", i)
	}
	fmt.Println("")

	add, del, replace := set2.Update(diff)
	for _, r := range add {
		fmt.Printf("Add:     %+v\n", r.Track)
	}
	for _, r := range del {
		fmt.Printf("Del:     %+v\n", r.Track)
	}
	for _, r := range replace {
		fmt.Printf("Replace: %+v -> %+v\n", r.Old.Track, r.New.Track)
	}
}
