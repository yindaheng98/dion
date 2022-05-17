package router

import (
	"github.com/yindaheng98/dion/pkg/sxu/bridge"
	pb "github.com/yindaheng98/dion/proto"
)

type ProcessorFactory interface {
	NewProcessor(init *pb.ProceedTrack) bridge.Processor
}
