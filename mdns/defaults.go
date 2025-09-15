
package mdns

const (
	DefaultServiceType = "_dns._udp"

	QueryPluginName = "dnsmesh_mdns"
	AdvertisePluginName = "dnsmesh_mdns_advertise"


	AdvertisingPrefix = "meshdns "
	DefaultTTL uint32 = 120

	DefaultAddrsPerHost = 1
	DefaultAddrMode = IPv4Only
)
