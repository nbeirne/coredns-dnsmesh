package self

import (
	"context"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

type mockIface struct {
	addrs []net.Addr
	err   error
}

type mockAddr struct {
	network string
	addr    string
}

func (m *mockAddr) Network() string {
	return m.network
}

func (m *mockAddr) String() string {
	return m.addr
}

func (m *mockIface) Addrs() ([]net.Addr, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.addrs, nil
}

// mockNetInterfaces is a mock implementation of the GetNetInterfaces interface.
type mockNetInterfaces struct {
	interfaces []Interface
	error      error
}

func (m *mockNetInterfaces) Interfaces() ([]Interface, error) {
	return m.interfaces, m.error
}

func TestFindLocalIP(t *testing.T) {
	_, mask, _ := net.ParseCIDR("192.168.2.0/24")
	ipNet := &net.IPNet{IP: net.ParseIP("192.168.1.1"), Mask: mask.Mask}
	mock := &mockNetInterfaces{
		interfaces: []Interface{
			&mockIface{
				addrs: []net.Addr{
					ipNet,
				},
			},
		},
	}

	s := Self{getInterfaces: mock}

	// Test with an IP in the mock interface's subnet
	ips, err := s.findLocalIPs("192.168.1.100")
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}
	if ips[0].String() != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, but got %s", ips[0].String())
	}

	// Test with an IP not in the mock interface's subnet
	ips, err = s.findLocalIPs("10.0.0.100")
	if err != nil || len(ips) != 0 {
		t.Fatalf("Expected no IPs and no error, but got ips: %v, err: %v", ips, err)
	}
}

func TestFindLocalIPs(t *testing.T) {
	_, mask, _ := net.ParseCIDR("192.168.2.0/24")
	ipNet := &net.IPNet{IP: net.ParseIP("192.168.1.56"), Mask: mask.Mask}
	mock := &mockNetInterfaces{
		interfaces: []Interface{
			&mockIface{
				addrs: []net.Addr{ipNet},
			},
		},
	}
	s := Self{
		Zones:         []string{"example.org."},
		getInterfaces: mock,
	}

	// Create a new DNS request
	r := new(dns.Msg)
	r.SetQuestion("self.example.org.", dns.TypeA)

	// Create a new recorder
	rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "192.168.1.100"})

	// Call the ServeDNS method
	_, err := s.ServeDNS(context.Background(), rec, r)
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}

	// Check the response
	if rec.Msg.Answer[0].Header().Rrtype != dns.TypeA {
		t.Errorf("Expected answer type A, but got %d", rec.Msg.Answer[0].Header().Rrtype)
	}
	a := rec.Msg.Answer[0].(*dns.A)
	if !a.A.Equal(net.ParseIP("192.168.1.56")) {
		t.Errorf("Expected IP 192.168.1.56, but got %s", a.A.String())
	}
}

func TestServeDNSMultipleIPs(t *testing.T) {
	_, v4mask, _ := net.ParseCIDR("192.168.1.0/24")
	_, v6mask, _ := net.ParseCIDR("fd00::/64")
	_, v6linklocalmask, _ := net.ParseCIDR("fe80::/64")

	mock := &mockNetInterfaces{
		interfaces: []Interface{
			&mockIface{
				addrs: []net.Addr{
					&net.IPNet{IP: net.ParseIP("192.168.1.10"), Mask: v4mask.Mask},
					&net.IPNet{IP: net.ParseIP("192.168.1.11"), Mask: v4mask.Mask},
					&net.IPNet{IP: net.ParseIP("fd00::10"), Mask: v6mask.Mask},
					&net.IPNet{IP: net.ParseIP("fe80::1"), Mask: v6linklocalmask.Mask},
				},
			},
		},
	}
	s := Self{
		Zones:         []string{"example.org."},
		getInterfaces: mock,
	}

	t.Run("A query with multiple IPv4", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "192.168.1.200"})

		_, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if len(rec.Msg.Answer) != 2 {
			t.Fatalf("Expected 2 A records, but got %d", len(rec.Msg.Answer))
		}

		found10 := false
		found11 := false
		for _, ans := range rec.Msg.Answer {
			if a, ok := ans.(*dns.A); ok {
				if a.A.Equal(net.ParseIP("192.168.1.10")) {
					found10 = true
				}
				if a.A.Equal(net.ParseIP("192.168.1.11")) {
					found11 = true
				}
			}
		}

		if !found10 || !found11 {
			t.Errorf("Expected to find both 192.168.1.10 and 192.168.1.11 in response")
		}
	})

	t.Run("AAAA query with one IPv6", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeAAAA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "192.168.1.200"})

		_, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if len(rec.Msg.Answer) != 1 {
			t.Fatalf("Expected 1 AAAA record, but got %d", len(rec.Msg.Answer))
		}

		if rec.Msg.Answer[0].Header().Rrtype != dns.TypeAAAA {
			t.Errorf("Expected answer type AAAA, but got %d", rec.Msg.Answer[0].Header().Rrtype)
		}
		aaaa := rec.Msg.Answer[0].(*dns.AAAA)
		if !aaaa.AAAA.Equal(net.ParseIP("fd00::10")) {
			t.Errorf("Expected IP fd00::10, but got %s", aaaa.AAAA.String())
		}
	})

	t.Run("AAAA query from IPv6 remote", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeAAAA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "fd00::200"})

		_, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if len(rec.Msg.Answer) != 1 {
			t.Fatalf("Expected 1 AAAA record, but got %d", len(rec.Msg.Answer))
		}

		if rec.Msg.Answer[0].Header().Rrtype != dns.TypeAAAA {
			t.Errorf("Expected answer type AAAA, but got %d", rec.Msg.Answer[0].Header().Rrtype)
		}
		aaaa := rec.Msg.Answer[0].(*dns.AAAA)
		if !aaaa.AAAA.Equal(net.ParseIP("fd00::10")) {
			t.Errorf("Expected IP fd00::10, but got %s", aaaa.AAAA.String())
		}
	})
}

func TestServeDNSMultipleInterfaces(t *testing.T) {
	_, v4mask1, _ := net.ParseCIDR("192.168.1.0/24")
	_, v4mask2, _ := net.ParseCIDR("10.0.0.0/8")
	_, v6mask, _ := net.ParseCIDR("fd00:1::/64")

	mock := &mockNetInterfaces{
		interfaces: []Interface{
			&mockIface{
				addrs: []net.Addr{
					&net.IPNet{IP: net.ParseIP("192.168.1.10"), Mask: v4mask1.Mask},
				},
			},
			&mockIface{
				addrs: []net.Addr{
					&net.IPNet{IP: net.ParseIP("10.0.0.20"), Mask: v4mask2.Mask},
				},
			},
			&mockIface{
				addrs: []net.Addr{
					&net.IPNet{IP: net.ParseIP("fd00:1::30"), Mask: v6mask.Mask},
				},
			},
		},
	}
	s := Self{
		Zones:         []string{"example.org."},
		getInterfaces: mock,
	}

	t.Run("Request from first subnet", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "192.168.1.200"})

		_, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if len(rec.Msg.Answer) != 1 {
			t.Fatalf("Expected 1 A record, but got %d", len(rec.Msg.Answer))
		}
		a := rec.Msg.Answer[0].(*dns.A)
		if !a.A.Equal(net.ParseIP("192.168.1.10")) {
			t.Errorf("Expected IP 192.168.1.10, but got %s", a.A.String())
		}
	})

	t.Run("Request from second subnet", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "10.1.2.3"})

		_, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if len(rec.Msg.Answer) != 1 {
			t.Fatalf("Expected 1 A record, but got %d", len(rec.Msg.Answer))
		}
		a := rec.Msg.Answer[0].(*dns.A)
		if !a.A.Equal(net.ParseIP("10.0.0.20")) {
			t.Errorf("Expected IP 10.0.0.20, but got %s", a.A.String())
		}
	})

	t.Run("Request from IPv6 subnet", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeAAAA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "fd00:1::200"})

		_, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}

		if len(rec.Msg.Answer) != 1 {
			t.Fatalf("Expected 1 AAAA record, but got %d", len(rec.Msg.Answer))
		}
		aaaa := rec.Msg.Answer[0].(*dns.AAAA)
		if !aaaa.AAAA.Equal(net.ParseIP("fd00:1::30")) {
			t.Errorf("Expected IP fd00:1::30, but got %s", aaaa.AAAA.String())
		}
	})

	t.Run("Request from unknown subnet", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "172.16.0.100"})

		rcode, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if rcode != dns.RcodeNameError {
			t.Errorf("Expected rcode %d, but got %d", dns.RcodeNameError, rcode)
		}
		if rec.Msg.Rcode != dns.RcodeNameError {
			t.Errorf("Expected response message with rcode %d, but got %d", dns.RcodeNameError, rec.Msg.Rcode)
		}
	})
}

func TestServeDNSZoneMatching(t *testing.T) {
	_, ipNet, _ := net.ParseCIDR("192.168.1.1/24")
	mock := &mockNetInterfaces{
		interfaces: []Interface{
			&mockIface{
				addrs: []net.Addr{ipNet},
			},
		},
	}

	s := Self{
		Zones:         []string{"example.org."},
		getInterfaces: mock,
		Next: plugin.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
			// This is our "next" plugin. If called, it means self passed the request on.
			// We'll return a unique rcode to confirm it was called.
			return dns.RcodeNotZone, nil
		}),
	}

	t.Run("Query inside zone", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.org.", dns.TypeA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "192.168.1.100"})

		rcode, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if rcode != dns.RcodeSuccess {
			t.Errorf("Expected rcode %d for in-zone query, but got %d", dns.RcodeSuccess, rcode)
		}
		if len(rec.Msg.Answer) == 0 {
			t.Fatal("Expected an answer for in-zone query, but got none")
		}
	})

	t.Run("Query with exact zone", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("example.org.", dns.TypeA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "192.168.1.100"})

		rcode, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if rcode != dns.RcodeSuccess {
			t.Errorf("Expected rcode %d for in-zone query, but got %d", dns.RcodeSuccess, rcode)
		}
		if len(rec.Msg.Answer) == 0 {
			t.Fatal("Expected an answer for in-zone query, but got none")
		}
	})

	t.Run("Query outside zone", func(t *testing.T) {
		r := new(dns.Msg)
		r.SetQuestion("self.example.com.", dns.TypeA)
		rec := dnstest.NewRecorder(&test.ResponseWriter{RemoteIP: "192.168.1.100"})

		rcode, err := s.ServeDNS(context.Background(), rec, r)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		if rcode != dns.RcodeNotZone { // Check for the rcode from our mock 'Next' plugin
			t.Errorf("Expected rcode %d for out-of-zone query, but got %d", dns.RcodeNotZone, rcode)
		}
	})
}
