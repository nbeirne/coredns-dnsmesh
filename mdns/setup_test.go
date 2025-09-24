package mdns

import (
	"net"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/coredns/caddy"

	"github.com/nbeirne/coredns-dnsmesh/mdns/browser"
)

func TestQuerySetup(t *testing.T) {
	successCases := []string{
		`dnsmesh_mdns example.com {
			type sometype
			iface_bind_subnet 127.0.0.0/24
			ignore_self true
			filter .*
			address_mode only_ipv6
			addresses_per_host 1
			timeout 2s
			zone somezone
			attempts 3
			worker_count 4
		}`,
		`dnsmesh_mdns example.com`,
		`dnsmesh_mdns example.com {
		}`,
		`dnsmesh_mdns example.com {
			timeout 4m
		}`,
		`dnsmesh_mdns example.com {
			filter ".*[A-Z]+.*"
		}`,
		`dnsmesh_mdns example.com {
			address_mode only_ipv6
			address_mode only_ipv4
			address_mode prefer_ipv6
			address_mode prefer_ipv4
		}`,
		`dnsmesh_mdns example.com {
			ignore_self false
		}`,
	}

	failureCases := []string{
		`dnsmesh_mdns`,
		`dnsmesh_mdns {
			iface_bind_subnet 127.0.0.1
		}`,
		`dnsmesh_mdns example.com {
			iface_bind_subnet 127.0.0.1
		}`,
		`dnsmesh_mdns example.com {
			address_mode badmode
		}`,
		`dnsmesh_mdns example.com {
			addresses_per_host n
		}`,
		`dnsmesh_mdns example.com {
			ignore_self
		}`,
		`dnsmesh_mdns example.com {
			ignore_self n
		}`,
		`dnsmesh_mdns example.com {
			timeout 1
		}`,
		`dnsmesh_mdns example.com {
			timeout n
		}`,
		`dnsmesh_mdns example.com {
			timeout
		}`,
		`dnsmesh_mdns example.com {
			worker_count
		}`,
		`dnsmesh_mdns example.com {
			worker_count n
		}`,
		`dnsmesh_mdns example.com {
			filter
		}`,
		`dnsmesh_mdns example.com {
			unknown
		}`,
	}

	for _, str := range successCases {
		c := caddy.NewTestController("dns", str)
		if err := setupQuery(c); err != nil {
			t.Fatalf("Expected no errors, but got: %v when parsing %s", err, str)
		}
	}

	for _, str := range failureCases {
		c := caddy.NewTestController("dns", str)
		if err := setupQuery(c); err == nil {
			t.Fatalf("Expected error, but got success when parsing %s", str)
		}
	}

	// check values of the full test case
	mockIfaces := []net.Interface{{Name: "mock0"}}
	mockFinder := func(subnet net.IPNet) ([]net.Interface, error) {
		return mockIfaces, nil
	}

	c := caddy.NewTestController("dns", successCases[0])
	meshdns, err := parseQueryOptions(c, mockFinder)
	parsedBrowser := meshdns.browser.(*browser.ZeroconfBrowser)
	if err != nil {
		t.Fatalf("parseQueryOptions failed for a success case: %v", err)
	}

	expectedBrowser := browser.NewZeroconfBrowser("local.", "sometype", &mockIfaces)
	expectedResult := MdnsMeshPlugin{
		browser:      expectedBrowser,
		ignoreSelf:   true,
		filter:       regexp.MustCompile(".*"),
		addrMode:     IPv6Only,
		addrsPerHost: 1,
		Timeout:      time.Second * 2,
		Zone:         "somezone",
		Attempts:     3,
		WorkerCount:  4,
	}

	// Check the browser's configured values individually instead of using DeepEqual.
	if parsedBrowser.Service() != expectedBrowser.Service() {
		t.Errorf("Expected browser service to be '%s', got '%s'", expectedBrowser.Service(), parsedBrowser.Service())
	}
	if parsedBrowser.Domain() != expectedBrowser.Domain() {
		t.Errorf("Expected browser domain to be '%s', got '%s'", expectedBrowser.Domain(), parsedBrowser.Domain())
	}
	if !reflect.DeepEqual(parsedBrowser.Interfaces(), expectedBrowser.Interfaces()) {
		t.Errorf("Expected browser interfaces to be %+v, got %+v", expectedBrowser.Interfaces(), parsedBrowser.Interfaces())
	}

	// Since browser's configuration is checked, we can nil it out for the rest of the struct comparison.
	meshdns.browser = nil // a more robust comparison would be to compare all other fields individually.
	expectedResult.browser = nil

	if expectedResult.ignoreSelf != meshdns.ignoreSelf {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.ignoreSelf, meshdns.ignoreSelf)
	}

	if !reflect.DeepEqual(expectedResult.filter, meshdns.filter) {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.filter, meshdns.filter)
	}

	if expectedResult.addrMode != meshdns.addrMode {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.addrMode, meshdns.addrMode)
	}

	if expectedResult.addrsPerHost != meshdns.addrsPerHost {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.addrsPerHost, meshdns.addrsPerHost)
	}

	if expectedResult.Timeout != meshdns.Timeout {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.Timeout, meshdns.Timeout)
	}

	if expectedResult.Zone != meshdns.Zone {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.Zone, meshdns.Zone)
	}

	if expectedResult.Attempts != meshdns.Attempts {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.Attempts, meshdns.Attempts)
	}

	if expectedResult.WorkerCount != meshdns.WorkerCount {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.WorkerCount, meshdns.WorkerCount)
	}
}

func TestAdvertiseSetup(t *testing.T) {
	// Corrected plugin name from meshdns-mdns-advertise to dnsmesh_mdns_advertise
	successCases := []string{
		`dnsmesh_mdns_advertise {
			instance_name testname
			type  testtype
			port 100
			ttl 100
			iface_bind_subnet 127.0.0.0/24
		}`,
		`dnsmesh_mdns_advertise`,
		`dnsmesh_mdns_advertise {
		}`,
	}

	failureCases := []string{
		//`meshdns-mdns-advertise example.com`,
		//`meshdns-mdns-advertise example.com {
		//}`,
		`dnsmesh_mdns_advertise {
			iface_bind_subnet 127.0.0.1
		}`,
		`dnsmesh_mdns_advertise {
			port m
		}`,
		`dnsmesh_mdns_advertise {
			ttl 1m
		}`,
	}

	for _, str := range successCases {
		c := caddy.NewTestController("dns", str)
		if err := setupAdvertise(c); err != nil {
			t.Fatalf("Expected no errors, but got: %v when parsing %s", err, str)
		}
	}

	for _, str := range failureCases {
		c := caddy.NewTestController("dns", str)
		if err := setupAdvertise(c); err == nil {
			t.Fatalf("Expected error, but got success when parsing %s", str)
		}
	}
}
