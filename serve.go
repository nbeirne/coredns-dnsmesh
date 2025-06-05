package dnsmesh

import (
	"context"
	"time"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/fanout"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("dnsmesh")

func (t *DnsMesh) CreateFanout() *fanout.Fanout {
	f := &fanout.Fanout {
		//TlsConfig             *tls.Config
		//ExcludeDomains        Domain
		//TlsServerName         string
		//LoadFactor            []int

		// TODO: init workers properly
		Timeout: 30 * time.Second, // TODO: const
		ExcludeDomains:        fanout.NewDomain(),
		Race: false,
		Net: fanout.Udp,
		From:                 t.zone, 
		Attempts:             3,
		ServerSelectionPolicy: &fanout.SequentialPolicy{},
		//TapPlugin:            *dnstap.Dnstap, // TODO: ??
		Next:                 t.next,
	}

	for _, provider := range t.mesh_providers {
		hosts := provider.mesh_dns_hosts()
		for _, host := range hosts {
			log.Infof("Add client: %s", host.Location.String())
			f.AddClient(fanout.NewClient(host.Location.String(), fanout.Udp))
		}
	}

	// TODO: set max workers... 

	return f
}

func (t *DnsMesh) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Infof("Received request for name: %v", r.Question[0].Name)

	f := t.CreateFanout()

	return f.ServeDNS(ctx, w, r)
}

