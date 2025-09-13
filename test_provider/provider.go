package test_provider

import (
	"github.com/nbeirne/coredns-dnsmesh/util"

	"context"
	"net/netip"

	//"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

type TestProvider struct {
	dnsMesh util.DnsMesh
}

var log = clog.NewWithPlugin("dnsmesh-test-provider")

// Name implements the Handler interface.
func (t *TestProvider) Name() string { return "dnsmesh-test-provider" }

func (t *TestProvider) Start() error {
	t.dnsMesh.AddMeshProvider(t)
	return nil
}

func (t *TestProvider) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Infof("Received request for name: %v", r.Question[0].Name)

	f := t.dnsMesh.CreateFanout()

	return f.ServeDNS(ctx, w, r)
}

func (t *TestProvider) MeshDnsHosts() []util.DnsHost {
	log.Infof("create mesh host.. providers: %s", t.dnsMesh.MeshProviders)
	return []util.DnsHost{
		{ Location: netip.MustParseAddrPort("127.0.0.1:1053") },
	}
}

