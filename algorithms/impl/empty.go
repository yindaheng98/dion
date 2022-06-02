package impl

import pb "github.com/yindaheng98/dion/proto"

type EmptyAlgorithm struct {
}

func (EmptyAlgorithm) UpdateSFUStatus(current []*pb.SFUStatus, reports []*pb.QualityReport) (expected []*pb.SFUStatus) {
	return current
}
