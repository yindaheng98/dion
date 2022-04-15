package algorithms

import pb "github.com/yindaheng98/isglb/proto"

type Algorithm interface {

	// UpdateSFUStatus tell the algorithm that the SFU graph has changed
	// The output is that which SFU's status should change
	UpdateSFUStatus(*pb.SFUStatus) []*pb.SFUStatus

	// GetSFUStatus get the expected SFU status from the algorithm
	GetSFUStatus(string) *pb.SFUStatus
}
