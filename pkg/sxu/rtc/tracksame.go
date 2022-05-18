package rtc

import (
	"github.com/pion/webrtc/v3"
	pb "github.com/yindaheng98/dion/proto"
)

var Layers = map[pb.Subscription_Layer]string{
	pb.Subscription_Q: "q",
	pb.Subscription_H: "h",
	pb.Subscription_F: "f",
}

func TrackSame(t1 *pb.Subscription, t2 *webrtc.TrackRemote) bool {
	if t1.TrackId != t2.ID() {
		return false
	}
	if Layers[t1.Layer] != t2.RID() {
		return false
	}
	return true
}

func (r *RTC) update(tracks []*pb.Subscription) error {
	trackInfos := make([]*Subscription, len(tracks))
	for i, track := range tracks {
		trackInfos[i] = &Subscription{
			TrackId:   track.TrackId,
			Mute:      track.Mute,
			Subscribe: true,
			Layer:     Layers[track.Layer],
		}
	}
	return r.Subscribe(trackInfos)
}

func (r *RTC) isSame(tracks []*pb.Subscription) bool {
	temp := map[string]*webrtc.TrackRemote{}
	for _, t := range r.peer.peer.Publisher().PublisherTracks() {
		temp[t.Track.ID()] = t.Track
	}
	for _, sub := range tracks {
		if t, ok := temp[sub.TrackId]; ok {
			if !TrackSame(sub, t) {
				return false
			}
		}
	}
	return true
}

func (r *RTC) Update(tracks []*pb.Subscription) error {
	if !r.isSame(tracks) {
		return r.update(tracks)
	}
	return nil
}
