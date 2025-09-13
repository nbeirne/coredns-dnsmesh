package util

import (
	"time"

	"net/netip"

	"github.com/coredns/coredns/plugin"
	"github.com/networkservicemesh/fanout"
)


type DnsHost struct {
	Location netip.AddrPort
	// TODO: add TLS info, if relevant.
}

func (d DnsHost) Compare(d2 DnsHost) int {
	return d.Location.Compare(d2.Location)
}


type MeshProvider interface{
	Start() error
	MeshDnsHosts() []DnsHost
}

type DnsMesh struct {
	Zone string
	Next plugin.Handler

	MeshProviders []MeshProvider
}

func (d *DnsMesh) AddMeshProvider(meshProvider MeshProvider) {
	d.MeshProviders = append(d.MeshProviders, meshProvider)
}

func (d *DnsMesh) CreateFanout() *fanout.Fanout {
	f := &fanout.Fanout {
		Timeout: 30 * time.Second, // TODO: const
		ExcludeDomains: fanout.NewDomain(),
		Race: false,  // TODO: we may actually want to race. 
		From: d.Zone, // TODO: what does d.zone do??
		Attempts: 3,
		ServerSelectionPolicy: &fanout.SequentialPolicy{},
		Next: d.Next,
		//ExcludeDomains        Domain
		// TODO: init workers properly
		//TapPlugin:            *dnstap.Dnstap, // TODO: setup tap plugin
	}

	for _, provider := range d.MeshProviders {
		//provider.MeshDnsHosts()
		hosts := provider.MeshDnsHosts()
		for _, host := range hosts {
			f.AddClient(fanout.NewClient(host.Location.String(), fanout.UDP))
		}
	}

	// TODO: set max workers... 
	return f
}

