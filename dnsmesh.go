package dnsmesh

import (
	"net/netip"

	"github.com/coredns/coredns/plugin"
)


type DnsHost struct {
	Location netip.AddrPort
	// TODO: add TLS info, if relevant.
}

func (d DnsHost) Compare(d2 DnsHost) int {
	return d.Location.Compare(d2.Location)
}


type MeshProvider interface{
	start() error
	mesh_dns_hosts() []DnsHost
}

type DnsMesh struct {
	zone string

	mesh_providers []MeshProvider
	next plugin.Handler
}

// Name implements the Handler interface.
func (d *DnsMesh) Name() string { return "dnsmesh" }

func (d *DnsMesh) start() error {
	for _, provider := range d.mesh_providers {
		err := provider.start()
		if (err != nil) {
			return err
		}
	}

	return nil
}
