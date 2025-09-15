package mdns

import (
	"github.com/nbeirne/coredns-dnsmesh/util"

	"context"
	"net"
	"net/netip"
	"regexp"

	"github.com/miekg/dns"
	clog "github.com/coredns/coredns/plugin/pkg/log"

	"github.com/celebdor/zeroconf"
)

type MdnsProvider struct {
	dnsMesh util.DnsMesh

	browser       *MdnsBrowser

	filter        *regexp.Regexp
	addrMode       int
	addrsPerHost   int
}

const (
	PreferIPv6 int = 0
	PreferIPv4     = 1
	IPv6Only	   = 2
	IPv4Only 	   = 3
)

// TODO: configure all settings
// TODO: fanout settings


var log = clog.NewWithPlugin("dnsmesh_mdns")

// Name implements the Handler interface.
func (m *MdnsProvider) Name() string { return QueryPluginName }

func (m *MdnsProvider) Start() error {
	log.Infof("Starting meshdns: %w", m.filter)

	m.dnsMesh.AddMeshProvider(m)

	m.browser.Start()

	return nil
}

func (m *MdnsProvider) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Infof("Received request for name: %v", r.Question[0].Name)

	f := m.dnsMesh.CreateFanout()
	return f.ServeDNS(ctx, w, r)
}

func (m *MdnsProvider) MeshDnsHosts() (outputHosts []util.DnsHost) {
	services := m.browser.Services()

	for _, service := range services {
		hosts := m.hostsForZeroconfServiceEntry(service)
		for _, host := range hosts {
			log.Infof("Found host for instance %s: %s", service.Instance, host.String())
			outputHosts = append(outputHosts, util.DnsHost{Location: host})
		}
	}

	return outputHosts
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

		// Ignore "self" IPs
		//iface, err := util.FindInterfaceForAddress(ip)
		//if err == nil {
		//	log.Debugf("Ignoring entry '%s' because the interface %s has the ip %s",
		//		entry.Instance, iface.Name, ip.String())
		//	continue // Skip this IP, it's local
		//}

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

