package mdns

import (
	"time"
)

const (
	DefaultServiceType = "_dns._udp"
	DefaultDomain      = "local."

	ForwardPluginName   = "dnsmesh_mdns_forward"
	AdvertisePluginName = "dnsmesh_mdns_advertise"

	AdvertisingPrefix        = "meshdns-"
	DefaultTTL        uint32 = 320

	DefaultTimeout      time.Duration = time.Second * 30
	DefaultAddrsPerHost               = 1
	DefaultAddrMode                   = IPv4Only
)
