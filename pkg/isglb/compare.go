package isglb

import pb "github.com/yindaheng98/dion/proto"

// SFUStatusIsSame compare whether the two SFUStatus is same
func SFUStatusIsSame(s1, s2 *pb.SFUStatus) bool {
	// TODO: ForwardTracks, ProceedTracks and ClientNeededSession maybe disorder
	return s1.String() == s2.String()
}
