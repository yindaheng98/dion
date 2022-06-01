package config

import "time"

const (
	ServiceISGLB         = "isglb"
	ServiceSXU           = "rtc"
	ServiceStupid        = "rtc"
	ServiceNameStupid    = "stupid"
	ServiceSessionStupid = "stupid"
	ServiceClient        = "client"

	DiscoveryExpire    = 500 * time.Millisecond
	DiscoveryLifeCycle = 200 * time.Millisecond

	ClientSessionExpire    = 500 * time.Millisecond
	ClientSessionLifeCycle = 200 * time.Millisecond
)
