package random

import (
	"fmt"
	"testing"

	pb "github.com/yindaheng98/dion/proto"
)

func TestRandom_UpdateSFUStatus(t *testing.T) {
	alg := &Random{}
	tr := &RandTransmissionReport{}
	cr := &RandComputationReport{}
	// alg.RandomTrack = true // set true to modify track list
	lst := make([]*pb.SFUStatus, 0)
	for i := 0; i < 100; i++ {
		reports := make([]*pb.QualityReport, 8)
		for j := 0; j < 2; j++ {
			reports[j*2] = &pb.QualityReport{Report: &pb.QualityReport_Transmission{Transmission: tr.RandReport()}}
			reports[j*2+1] = &pb.QualityReport{Report: &pb.QualityReport_Computation{Computation: cr.RandReport()}}
		}
		lst = alg.UpdateSFUStatus(lst, reports)
	}
}

func TestRandReports_RandReports(t *testing.T) {
	tr := &RandTransmissionReport{}
	cr := &RandComputationReport{}
	for i := 0; i < 100; i++ {
		fmt.Printf("%+v\n", &pb.QualityReport{Report: &pb.QualityReport_Transmission{Transmission: tr.RandReport()}})
		fmt.Printf("%+v\n", &pb.QualityReport{Report: &pb.QualityReport_Computation{Computation: cr.RandReport()}})
	}
}
