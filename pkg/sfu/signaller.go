package sfu

import "github.com/pion/ion/proto/rtc"

type Signaller struct {
	sig rtc.RTC_SignalServer
}

func NewSignaller(sig rtc.RTC_SignalServer) Signaller {
	return Signaller{sig: sig}
}
