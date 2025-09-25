package mdns

import (
	"context"
	"net"
	"net/netip"
	"regexp"
	"time"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/miekg/dns"

	"github.com/grandcat/zeroconf"

	"github.com/coredns/coredns/plugin"
	"github.com/networkservicemesh/fanout"

	"github.com/nbeirne/coredns-dnsmesh/mdns/browser"
)

const (
	PreferIPv6 int = 0
	PreferIPv4     = 1
	IPv6Only       = 2
	IPv4Only       = 3
)

type MdnsForwardPlugin struct {
	// fanout
	Timeout     time.Duration  // overall timeout for a whole request
	Zone        string         // only process requests to this domain
	Attempts    int            // attempts per server
	WorkerCount int            // number of requests to run in parallel
	Next        plugin.Handler // next plugin if req not in zone or it is an excluded domains
	//ExcludeDomains        Domain 			  // TODO??  exclude domains from the fanout
	//ServerSelectionPolicy 	 			  // TODO??  select which servers are requested first
	// TODO: fallthrough on error?

	// internal filters
	filter       *regexp.Regexp
	ignoreSelf   bool
	addrMode     int
	addrsPerHost int

	browser browser.MdnsBrowserInterface
}

// TODO: fanout settings

var log = clog.NewWithPlugin(ForwardPluginName)

// Name implements the Handler interface.
func (m *MdnsForwardPlugin) Name() string { return ForwardPluginName }

func (m *MdnsForwardPlugin) Start() error {
	log.Infof("Starting meshdns...")

	m.browser.Start()

	return nil
}

func (m *MdnsForwardPlugin) CreateFanout() *fanout.Fanout {
	f := &fanout.Fanout{
		Timeout:               m.Timeout,
		ExcludeDomains:        fanout.NewDomain(), // TODO - no excludes
		Race:                  false,              // first to respond wins, even if !success
		From:                  m.Zone,
		Attempts:              m.Attempts,
		ServerSelectionPolicy: &fanout.SequentialPolicy{},
		Next:                  m.Next,
		WorkerCount:           m.WorkerCount, // TODO: init workers properly
		//TapPlugin:            *dnstap.Dnstap, // TODO: setup tap plugin
	}

	services := m.browser.Services()
	for _, service := range services {
		hosts := m.hostsForZeroconfServiceEntry(service)
		for _, host := range hosts {
			log.Infof("Found host for instance %s: %s", service.Instance, host.String())
			f.AddClient(fanout.NewClient(host.String(), fanout.UDP))
		}
	}

	return f
}

func (m *MdnsForwardPlugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Debugf("Received request for name: %v", r.Question[0].Name)

	f := m.CreateFanout()
	return f.ServeDNS(ctx, w, r)
}

func (m *MdnsForwardPlugin) hostsForZeroconfServiceEntry(entry *zeroconf.ServiceEntry) (hosts []netip.AddrPort) {
	if m.filter != nil && !m.filter.MatchString(entry.Instance) {
		log.Debugf("Ignoring entry '%s' because the instance name did not match the filter: '%s'",
			entry.Instance, m.filter.String())
		return []netip.AddrPort{}
	}

	ips := []net.IP{}
	switch m.addrMode {
	case PreferIPv6:
		ips = append(ips, entry.AddrIPv6...)
		ips = append(ips, entry.AddrIPv4...)
	case PreferIPv4:
		ips = append(ips, entry.AddrIPv4...)
		ips = append(ips, entry.AddrIPv6...)
	case IPv6Only:
		ips = append(ips, entry.AddrIPv6...)
	case IPv4Only:
		ips = append(ips, entry.AddrIPv4...)
	}

	for idx, ip := range ips {
		if m.addrsPerHost > 0 && idx >= m.addrsPerHost {
			break
		}

		if m.ignoreSelf {
			iface, err := FindInterfaceForAddress(ip)
			if err == nil {
				log.Debugf("Ignoring entry '%s' because the interface %s has the ip %s",
					entry.Instance, iface.Name, ip.String())
				continue // Skip this IP, it's local
			}
		}

		addr, ok := netip.AddrFromSlice(ip)
		port := uint16(entry.Port)
		if !ok {
			log.Warningf("Ignoring address for entry '%s' because it could not be parsed: %s",
				entry.Instance, ip.String())
			continue
		}
		hosts = append(hosts, netip.AddrPortFrom(addr, port))
	}

	return hosts
}
