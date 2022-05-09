package rtc

import (
	"github.com/pion/webrtc/v3"
	pb "github.com/yindaheng98/dion/proto"
)

func TrackSame(t1 *pb.Subscription, t2 *webrtc.TrackRemote) bool {
	if t1.TrackId != t2.ID() {
		return false
	}
	if Layers[t1.Layer] != t2.RID() {
		return false
	}
	return true
}
