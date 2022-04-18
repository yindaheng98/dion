package algorithms

import pb "github.com/yindaheng98/isglb/proto"

// Algorithm is the node selection algorithm interface
type Algorithm interface {

	// UpdateSFUStatus tell the algorithm that the SFU graph and the computation and communication quality has changed
	// `current` is the changed SFU's current status
	// `reports` is a series of Quality Report
	// `expected` is that which SFU's status should change
	UpdateSFUStatus(current []*pb.SFUStatus, reports []*pb.QualityReport) (expected []*pb.SFUStatus)
}
