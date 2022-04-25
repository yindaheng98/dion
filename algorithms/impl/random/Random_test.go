package random

import (
	"fmt"
	pb "github.com/yindaheng98/isglb/proto"
	"testing"
)

func TestRandom_UpdateSFUStatus(t *testing.T) {
	alg := &Random{}
	rr := &RandReports{}
	// alg.RandomTrack = true // set true to modify track list
	lst := make([]*pb.SFUStatus, 0)
	for i := 0; i < 100; i++ {
		lst = alg.UpdateSFUStatus(lst, rr.RandReports())
	}
}

func TestRandReports_RandReports(t *testing.T) {
	rr := &RandReports{}
	for i := 0; i < 100; i++ {
		fmt.Printf("%+v\n", rr.RandReports())
	}
}
