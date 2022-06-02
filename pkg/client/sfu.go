package client

import (
	"github.com/pion/webrtc/v3"
	"github.com/yindaheng98/dion/pkg/sfu"
	pb2 "github.com/yindaheng98/dion/proto"
	"sync"
	"sync/atomic"
)

type Subscriber struct {
	client  *sfu.Client
	session *pb2.ClientNeededSession
	OnTrack func(remote *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)

	pc             atomic.Value
	recvCandidates []webrtc.ICECandidateInit
	recvCandMu     sync.Mutex
}
