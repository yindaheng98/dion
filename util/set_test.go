package util

import (
	"fmt"
	pb "github.com/yindaheng98/isglb/proto"
	"math/rand"
	"strconv"
	"testing"
)

func randTracks(n int) []*pb.ProceedTrack {
	ts := make([]*pb.ProceedTrack, n)
	for i := 0; i < n; i++ {
		ts[i] = &pb.ProceedTrack{
			SrcTrackId: strconv.Itoa(rand.Intn(100)),
			DstTrackId: strconv.Itoa(rand.Intn(100)),
		}
	}
	return ts
}

const N = 10

func TestIndex(t *testing.T) {
	index := NewIndex()
	templates := randTracks(N)
	for i := 0; i < 10; i++ {
		index.Add(proceedIndexData{
			proceedTrack: templates[i],
		})
	}
	fmt.Printf("%+v\n", IndexDataList(index.Gather()).ToProceedTracks())
	for i := 0; i < N/2; i++ {
		index.Del(proceedIndexData{
			proceedTrack: templates[rand.Int31n(N)],
		})
	}
	fmt.Printf("%+v\n", IndexDataList(index.Gather()).ToProceedTracks())

	index2 := NewIndex()
	index2.Del(proceedIndexData{
		proceedTrack: templates[rand.Int31n(N)],
	})
	for i := 0; i < N/2; i++ {
		index2.Add(proceedIndexData{
			proceedTrack: templates[rand.Int31n(N)],
		})
	}
	fmt.Printf("index: %+v\n", IndexDataList(index2.Gather()).ToProceedTracks())
	diff := make([]IndexData, N/2)
	for i := 0; i < N/2; i++ {
		diff[i] = proceedIndexData{
			proceedTrack: templates[rand.Int31n(N)],
		}
		diff[i].(proceedIndexData).proceedTrack.SrcTrackId = strconv.Itoa(rand.Intn(100))
	}
	fmt.Printf("updat: %+v\n", IndexDataList(diff).ToProceedTracks())
	add, del, replace := index2.Update(diff)
	fmt.Printf("Add: %+v\n", IndexDataList(add).ToProceedTracks())
	fmt.Printf("Del: %+v\n", IndexDataList(del).ToProceedTracks())
	for _, r := range replace {
		fmt.Printf("%+v -> %+v\n", r.Old.(proceedIndexData).proceedTrack, r.New.(proceedIndexData).proceedTrack)
	}
}
