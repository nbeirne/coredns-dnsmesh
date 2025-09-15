package mdns

import (
	"context"
	"net"
	"net/netip"
	"time"
	"regexp"

	"github.com/miekg/dns"
	clog "github.com/coredns/coredns/plugin/pkg/log"

	"github.com/celebdor/zeroconf"

	"github.com/coredns/coredns/plugin"
	"github.com/networkservicemesh/fanout"
)

const (
	PreferIPv6 int = 0
	PreferIPv4     = 1
	IPv6Only	   = 2
	IPv4Only 	   = 3
)

type MdnsProvider struct {
	// fanout
	Timeout               time.Duration 	// overall timeout for a whole request
	Zone                  string 			// only process requests to this domain
	Attempts              int 				// attempts per server
	WorkerCount           int 				// number of requests to run in parallel
	Next                  plugin.Handler    // next plugin if req not in zone or it is an excluded domains 
	//ExcludeDomains        Domain 			  // TODO??  exclude domains from the fanout
	//ServerSelectionPolicy 	 			  // TODO??  select which servers are requested first
	// TODO: fallthrough on error?


	// internal filters
	filter        *regexp.Regexp
	ignoreSelf 	   bool
	addrMode       int
	addrsPerHost   int

	browser       *MdnsBrowser
}

// TODO: fanout settings


var log = clog.NewWithPlugin("dnsmesh_mdns")

// Name implements the Handler interface.
func (m *MdnsProvider) Name() string { return QueryPluginName }

func (m *MdnsProvider) Start() error {
	log.Infof("Starting meshdns...")

	m.browser.Start()

	return nil
}

func (m *MdnsProvider) CreateFanout() *fanout.Fanout {
	f := &fanout.Fanout {
		Timeout: m.Timeout * time.Second,
		ExcludeDomains: fanout.NewDomain(), // TODO - no excludes
		Race: false,  // first to respond wins, even if !success
		From: m.Zone,
		Attempts: m.Attempts,
		ServerSelectionPolicy: &fanout.SequentialPolicy{},
		Next: m.Next,
		WorkerCount: m.WorkerCount,
		// TODO: init workers properly
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

	// TODO: set max workers... 
	return f
}

func (m *MdnsProvider) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Infof("Received request for name: %v", r.Question[0].Name)

	f := m.CreateFanout()
	return f.ServeDNS(ctx, w, r)
}


func (m *MdnsProvider) hostsForZeroconfServiceEntry(entry *zeroconf.ServiceEntry) (hosts []netip.AddrPort) {
	if m.filter != nil && !m.filter.MatchString(entry.Instance) {
		log.Errorf("Ignoring entry '%s' because the instance name did not match the filter: '%s'",
				entry.Instance, m.filter.String())
		return []netip.AddrPort{}
	}

	ips := []net.IP{}
	if m.addrMode == PreferIPv6 {
		ips = append(ips, entry.AddrIPv6...)
		ips = append(ips, entry.AddrIPv4...)
	} else if m.addrMode == PreferIPv4 {
		ips = append(ips, entry.AddrIPv4...)
		ips = append(ips, entry.AddrIPv6...)
	} else if m.addrMode == IPv6Only {
		ips = append(ips, entry.AddrIPv6...)
	} else if m.addrMode == IPv4Only {
		ips = append(ips, entry.AddrIPv4...)
	}

	for idx, ip := range ips {
		if idx >= m.addrsPerHost {
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
			log.Errorf("Ignoring entry '%s' because the address was not able to be parsed",
				entry.Instance)
			continue
		}
		hosts = append(hosts, netip.AddrPortFrom(addr, port))
	}

	return hosts
}

