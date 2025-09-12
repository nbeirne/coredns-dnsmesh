package dnsmesh

import (
	"context"
	"time"

	"net/netip"

	"github.com/coredns/coredns/plugin"
	"github.com/networkservicemesh/fanout"
	"github.com/miekg/dns"
	clog "github.com/coredns/coredns/plugin/pkg/log"
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

	meshProviders []MeshProvider
}

func (d *DnsMesh) AddMeshProvider(meshProvider MeshProvider) {
	d.meshProviders = append(d.meshProviders, meshProvider)
}

var log = clog.NewWithPlugin("dnsmesh")

// Name implements the Handler interface.
func (d *DnsMesh) Name() string { return "dnsmesh" }

func (d *DnsMesh) Start() error {
	for _, provider := range d.meshProviders {
		err := provider.Start()
		if (err != nil) {
			return err
		}
	}

	return nil
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

	for _, provider := range d.meshProviders {
		//provider.MeshDnsHosts()
		hosts := provider.MeshDnsHosts()
		for _, host := range hosts {
			log.Infof("Add client: %s", host.Location.String())
			f.AddClient(fanout.NewClient(host.Location.String(), fanout.UDP))
		}
	}

	// TODO: set max workers... 

	return f
}

func (d *DnsMesh) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Infof("Received request for name: %v", r.Question[0].Name)

	f := d.CreateFanout()

	return f.ServeDNS(ctx, w, r)
}

