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

func TestQuerySetupSuccess(t *testing.T) {
	mockIfaces := []net.Interface{{Name: "mock0"}}
	mockFinder := func(subnet net.IPNet) ([]net.Interface, error) {
		return mockIfaces, nil
	}

	testCases := []struct {
		name           string
		input          string
		expectedPlugin *MdnsForwardPlugin
	}{
		{
			name: "full configuration",
			input: `dnsmesh_mdns example.com {
			type sometype
			iface_bind_subnet 127.0.0.0/24
			ignore_self true
			filter .*
			address_mode only_ipv6
			addresses_per_host 1
			timeout 5s
			attempts 3
			worker_count 4
		}`,
			expectedPlugin: &MdnsForwardPlugin{
				browser:      browser.NewZeroconfBrowser("local.", "sometype", &mockIfaces),
				ignoreSelf:   true,
				filter:       regexp.MustCompile(".*"),
				addrMode:     IPv6Only,
				addrsPerHost: 1,
				Timeout:      5 * time.Second,
				Zone:         "example.com",
				Attempts:     3,
				WorkerCount:  4,
			},
		},
		{
			name:  "minimal config",
			input: `dnsmesh_mdns example.com`,
			expectedPlugin: &MdnsForwardPlugin{
				browser:      browser.NewZeroconfBrowser("local.", DefaultServiceType, nil),
				addrMode:     DefaultAddrMode,
				addrsPerHost: DefaultAddrsPerHost,
				Timeout:      DefaultTimeout,
				Zone:         "example.com",
			},
		},
		{
			name:  "empty block",
			input: `dnsmesh_mdns example.com {}`,
			expectedPlugin: &MdnsForwardPlugin{
				browser:      browser.NewZeroconfBrowser("local.", DefaultServiceType, nil),
				addrMode:     DefaultAddrMode,
				addrsPerHost: DefaultAddrsPerHost,
				Timeout:      DefaultTimeout,
				Zone:         "example.com",
			},
		},
		{
			name:  "custom timeout",
			input: `dnsmesh_mdns example.com { timeout 4m }`,
			expectedPlugin: &MdnsForwardPlugin{
				browser:      browser.NewZeroconfBrowser("local.", DefaultServiceType, nil),
				addrMode:     DefaultAddrMode,
				addrsPerHost: DefaultAddrsPerHost,
				Timeout:      4 * time.Minute,
				Zone:         "example.com",
			},
		},
		{
			name:  "custom filter with space",
			input: `dnsmesh_mdns example.com { filter ".*[A-Z] .*" }`,
			expectedPlugin: &MdnsForwardPlugin{
				browser:      browser.NewZeroconfBrowser("local.", DefaultServiceType, nil),
				filter:       regexp.MustCompile(".*[A-Z] .*"),
				addrMode:     DefaultAddrMode,
				addrsPerHost: DefaultAddrsPerHost,
				Timeout:      DefaultTimeout,
				Zone:         "example.com",
			},
		},
		{
			name: "address mode override",
			input: `dnsmesh_mdns example.com {
				address_mode only_ipv6
				address_mode only_ipv4
				address_mode prefer_ipv6
				address_mode prefer_ipv4
			}`,
			expectedPlugin: &MdnsForwardPlugin{
				browser:      browser.NewZeroconfBrowser("local.", DefaultServiceType, nil),
				addrMode:     PreferIPv4, // Last one wins
				addrsPerHost: DefaultAddrsPerHost,
				Timeout:      DefaultTimeout,
				Zone:         "example.com",
			},
		},
		{
			name:  "explicitly disable ignore_self",
			input: `dnsmesh_mdns example.com { ignore_self false }`,
			expectedPlugin: &MdnsForwardPlugin{
				browser:      browser.NewZeroconfBrowser("local.", DefaultServiceType, nil),
				ignoreSelf:   false,
				addrMode:     DefaultAddrMode,
				addrsPerHost: DefaultAddrsPerHost,
				Timeout:      DefaultTimeout,
				Zone:         "example.com",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := caddy.NewTestController("dns", tc.input)
			plugin, err := parseForwardOptions(c, mockFinder)

			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			assertQueryPluginsEqual(t, tc.expectedPlugin, plugin)
		})
	}
}

func TestQuerySetupFailure(t *testing.T) {
	mockFinder := func(subnet net.IPNet) ([]net.Interface, error) {
		return nil, nil
	}

	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "no zone",
			input: `dnsmesh_mdns`,
		},
		{
			name:  "bad subnet",
			input: `dnsmesh_mdns example.com { iface_bind_subnet 127.0.0.1 }`,
		},
		{
			name:  "bad address_mode",
			input: `dnsmesh_mdns example.com { address_mode badmode }`,
		},
		{
			name:  "bad timeout",
			input: `dnsmesh_mdns example.com { timeout 1 }`,
		},
		{
			name:  "unknown option",
			input: `dnsmesh_mdns example.com { unknown }`,
		},
		{
			name:  "bad addresses_per_host",
			input: `dnsmesh_mdns example.com { addresses_per_host n }`,
		},
		{
			name:  "missing ignore_self value",
			input: `dnsmesh_mdns example.com { ignore_self }`,
		},
		{
			name:  "bad ignore_self value",
			input: `dnsmesh_mdns example.com { ignore_self n }`,
		},
		{
			name:  "bad timeout value",
			input: `dnsmesh_mdns example.com { timeout n }`,
		},
		{
			name:  "missing timeout value",
			input: `dnsmesh_mdns example.com { timeout }`,
		},
		{
			name:  "missing worker_count value",
			input: `dnsmesh_mdns example.com { worker_count }`,
		},
		{
			name:  "bad worker_count value",
			input: `dnsmesh_mdns example.com { worker_count n }`,
		},
		{
			name:  "missing filter value",
			input: `dnsmesh_mdns example.com { filter }`,
		},
		{
			name:  "missing type value",
			input: `dnsmesh_mdns example.com { type }`,
		},
		{
			name:  "missing attempts value",
			input: `dnsmesh_mdns example.com { attempts }`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := caddy.NewTestController("dns", tc.input)
			_, err := parseForwardOptions(c, mockFinder)
			if err == nil {
				t.Fatal("Expected an error, but got none")
			}
		})
	}
}

func assertQueryPluginsEqual(t *testing.T, expected, actual *MdnsForwardPlugin) {
	t.Helper()

	// Compare browser fields separately
	expectedBrowser := expected.browser.(*browser.ZeroconfBrowser)
	actualBrowser := actual.browser.(*browser.ZeroconfBrowser)

	if expectedBrowser.Service() != actualBrowser.Service() {
		t.Errorf("Browser service mismatch: want %q, got %q", expectedBrowser.Service(), actualBrowser.Service())
	}
	if expectedBrowser.Domain() != actualBrowser.Domain() {
		t.Errorf("Browser domain mismatch: want %q, got %q", expectedBrowser.Domain(), actualBrowser.Domain())
	}
	if !reflect.DeepEqual(expectedBrowser.Interfaces(), actualBrowser.Interfaces()) {
		t.Errorf("Browser interfaces mismatch: want %+v, got %+v", expectedBrowser.Interfaces(), actualBrowser.Interfaces())
	}

	// Compare filter regex strings
	expectedFilter := ""
	if expected.filter != nil {
		expectedFilter = expected.filter.String()
	}
	actualFilter := ""
	if actual.filter != nil {
		actualFilter = actual.filter.String()
	}
	if expectedFilter != actualFilter {
		t.Errorf("Filter mismatch: want %q, got %q", expectedFilter, actualFilter)
	}

	// Nil out the fields we've already checked for the final DeepEqual
	expected.browser = nil
	actual.browser = nil
	expected.filter = nil
	actual.filter = nil

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Plugin mismatch:\n- Want: %+v\n- Got:  %+v", expected, actual)
	}
}

func TestAdvertiseSetupSuccess(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name: "full config",
			input: `dnsmesh_mdns_advertise {
			instance_name testname
			type  testtype
			port 100
			ttl 100
			iface_bind_subnet 127.0.0.0/24
		}`,
		},
		{name: "minimal config", input: `dnsmesh_mdns_advertise`},
		{name: "empty block", input: `dnsmesh_mdns_advertise {}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := caddy.NewTestController("dns", tc.input)
			err := setupAdvertise(c)
			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestAdvertiseSetupFailure(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{name: "bad subnet", input: `dnsmesh_mdns_advertise { iface_bind_subnet 127.0.0.1 }`},
		{name: "bad port", input: `dnsmesh_mdns_advertise { port m }`},
		{name: "bad ttl", input: `dnsmesh_mdns_advertise { ttl 1m }`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := caddy.NewTestController("dns", tc.input)
			err := setupAdvertise(c)
			if err == nil {
				t.Fatal("Expected an error, but got none")
			}
		})
	}
}
