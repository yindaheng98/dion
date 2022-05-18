package signaller

import (
	"google.golang.org/grpc"
)

type ConnPool interface {
	GetConn(service, peerNID string) grpc.ClientConnInterface
}
