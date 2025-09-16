
package mdns

import (
	"time"
)

const (
	DefaultServiceType = "_dns._udp"

	QueryPluginName = "dnsmesh_mdns"
	AdvertisePluginName = "dnsmesh_mdns_advertise"


	AdvertisingPrefix = "meshdns "
	DefaultTTL uint32 = 120

	DefaultTimeout time.Duration = time.Second * 30
	DefaultAddrsPerHost = 1
	DefaultAddrMode = IPv4Only
)
