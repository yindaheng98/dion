package signal

import (
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/node/sfu"
)

func NewSFUService(SFU *ion_sfu.SFU) *sfu.SFUService {
	return sfu.NewSFUServiceWithSFU(SFU)
}
