package self

import (
	"context"
	"net"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

// Self is a plugin that returns the IP address of the server.
type Self struct {
	Next  plugin.Handler
	Zones []string

	getInterfaces GetNetInterfaces
}

func NewSelf(next plugin.Handler, zones []string) Self {
	return Self{
		Next:          next,
		Zones:         zones,
		getInterfaces: DefaultNetInterfacesImpl{},
	}
}

// ServeDNS implements the plugin.Handler interface.
func (s Self) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()

	zone := plugin.Zones(s.Zones).Matches(qname)

	if zone == "" {
		return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
	}

	remoteIP, _, err := net.SplitHostPort(w.RemoteAddr().String())
	if err != nil {
		log.Errorf("error parsing the repot IP: %v. Tried to parse %s", err, w.RemoteAddr().String())
		return dns.RcodeServerFailure, err
	}

	ips, err := s.findLocalIPs(remoteIP)
	if err != nil {
		log.Errorf("error finding server's IPs: %v", err)
		return dns.RcodeServerFailure, nil
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = make([]dns.RR, 0, len(ips))

	for _, ip := range ips {
		if ip.To4() != nil && state.QType() == dns.TypeA {
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600},
				A:   *ip,
			})
		} else if ip.To4() == nil && state.QType() == dns.TypeAAAA {
			m.Answer = append(m.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: qname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 3600},
				AAAA: *ip,
			})
		}
	}

	// Only write a response if we have answers
	if len(m.Answer) > 0 {
		w.WriteMsg(m)
		return dns.RcodeSuccess, nil
	}

	// If we are authoritative for the zone, but have no answer, we should return NXDOMAIN.
	// This prevents the query from being passed to the next plugin.
	m.Rcode = dns.RcodeNameError
	w.WriteMsg(m)
	return dns.RcodeNameError, nil
}

// Name implements the plugin.Handler interface.
func (s Self) Name() string { return "self" }

func (s Self) findLocalIPs(remoteIP string) ([]*net.IP, error) {
	ifaces, err := s.getInterfaces.Interfaces()
	if err != nil {
		return nil, err
	}

	parsedRemoteIP := net.ParseIP(remoteIP)
	ips := make([]*net.IP, 0)

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		// Check if any address on this interface is in the same subnet as the remote IP.
		foundMatch := false
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.Contains(parsedRemoteIP) {
					foundMatch = true
					break
				}
			}
		}

		// If we found a matching interface, collect all of its IPs and return.
		if foundMatch {
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok {
					if !ipNet.IP.IsLinkLocalUnicast() {
						ip := ipNet.IP
						ips = append(ips, &ip)
					}
				}
			}
		}
	}

	return ips, nil
}

// allow for mocking in tests
type GetNetInterfaces interface {
	Interfaces() ([]Interface, error)
}

type Interface interface {
	Addrs() ([]net.Addr, error)
}

type DefaultNetInterfacesImpl struct{}

func (d DefaultNetInterfacesImpl) Interfaces() ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]Interface, len(ifaces))
	for i := range ifaces {
		result[i] = &ifaces[i]
	}
	return result, nil
}
