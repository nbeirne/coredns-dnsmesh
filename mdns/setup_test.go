
package mdns 

import (
	"net"
	"time"
	"testing"
	"regexp"
	"reflect"

	"github.com/coredns/caddy"
)


func TestQuerySetup(t *testing.T) {
	successCases := []string {
		`meshdns-mdns example.com {
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
		`meshdns-mdns example.com`,
		`meshdns-mdns example.com {
		}`,
		`meshdns-mdns example.com {
			timeout 4m
		}`,
		`meshdns-mdns example.com {
			filter ".*[A-Z]+.*"
		}`,
		`meshdns-mdns example.com {
			address_mode only_ipv6
			address_mode only_ipv4
			address_mode prefer_ipv6
			address_mode prefer_ipv4
		}`,
		`meshdns-mdns example.com {
			ignore_self false
		}`,
	}

	failureCases := []string {
		`meshdns-mdns`,
		`meshdns-mdns {
			iface_bind_subnet 127.0.0.1
		}`,
		`meshdns-mdns example.com {
			iface_bind_subnet 127.0.0.1
		}`,
		`meshdns-mdns example.com {
			address_mode badmode
		}`,
		`meshdns-mdns example.com {
			addresses_per_host n
		}`,
		`meshdns-mdns example.com {
			ignore_self
		}`,
		`meshdns-mdns example.com {
			ignore_self n
		}`,
		`meshdns-mdns example.com {
			timeout 1
		}`,
		`meshdns-mdns example.com {
			timeout n
		}`,
		`meshdns-mdns example.com {
			timeout
		}`,
		`meshdns-mdns example.com {
			worker_count
		}`,
		`meshdns-mdns example.com {
			worker_count n
		}`,
		`meshdns-mdns example.com {
			filter
		}`,
		`meshdns-mdns example.com {
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
	c := caddy.NewTestController("dns", successCases[0])
	meshdns, _ := parseQueryOptions(c)

	_, subnet , _:= net.ParseCIDR("127.0.0.0/24")
	expectedResult := MdnsProvider{
		browser: &(MdnsBrowser {
			mdnsType: "sometype",
			ifaceBindSubnet: subnet,
		}),
		ignoreSelf: true,
		filter: regexp.MustCompile(".*"),
		addrMode: IPv6Only,
		addrsPerHost: 1,
		Timeout: time.Second * 2,
		Zone: "somezone",
		Attempts: 3,
		WorkerCount: 4,
	}

	if expectedResult.browser.mdnsType != meshdns.browser.mdnsType {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.browser.mdnsType, meshdns.browser.mdnsType)
	}

	if !reflect.DeepEqual(expectedResult.browser.ifaceBindSubnet, meshdns.browser.ifaceBindSubnet) {
		t.Fatalf("Expected results to be equal: %v %v", expectedResult.browser.ifaceBindSubnet, meshdns.browser.ifaceBindSubnet)
	}

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
	successCases := []string {
		`meshdns-mdns-advertise {
			instance_name testname
			type  testtype
			port 100
			ttl 100
			iface_bind_subnet 127.0.0.0/24
		}`,
		`meshdns-mdns-advertise`,
		`meshdns-mdns-advertise {
		}`,
	}

	failureCases := []string {
		//`meshdns-mdns-advertise example.com`,
		//`meshdns-mdns-advertise example.com {
		//}`,
		`meshdns-mdns-advertise {
			iface_bind_subnet 127.0.0.1
		}`,
		`meshdns-mdns-advertise {
			port m
		}`,
		`meshdns-mdns-advertise {
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

