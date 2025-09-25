package mdns

import (
	"net"
	"net/netip"
	"reflect"
	"regexp"
	"testing"

	"github.com/grandcat/zeroconf"
)

func mustParseAddrPorts(addrs ...string) []netip.AddrPort {
	ports := make([]netip.AddrPort, len(addrs))
	for i, s := range addrs {
		ports[i] = netip.MustParseAddrPort(s)
	}
	return ports
}

func TestHostFiltering(t *testing.T) {

	entry := zeroconf.ServiceEntry{
		ServiceRecord: zeroconf.ServiceRecord{Instance: "test_instance_name"},
		AddrIPv4: []net.IP{
			netip.MustParseAddr("127.0.0.1").AsSlice(),
			netip.MustParseAddr("2.2.2.2").AsSlice(),
			netip.MustParseAddr("3.3.3.3").AsSlice(),
		},
		AddrIPv6: []net.IP{
			net.ParseIP("::1"), // this is loopback in ipv6 land
			net.ParseIP("::2"),
			net.ParseIP("::3"),
		},
		Port: 10,
	}

	testCases := []struct {
		name     string
		plugin   MdnsMeshPlugin
		expected []netip.AddrPort
	}{
		{
			name:   "defaults (prefer ipv6)",
			plugin: MdnsMeshPlugin{addrMode: PreferIPv6},
			expected: mustParseAddrPorts(
				"[::1]:10", "[::2]:10", "[::3]:10",
				"127.0.0.1:10", "2.2.2.2:10", "3.3.3.3:10",
			),
		},
		{
			name:   "ipv6_only",
			plugin: MdnsMeshPlugin{addrMode: IPv6Only},
			expected: mustParseAddrPorts(
				"[::1]:10", "[::2]:10", "[::3]:10",
			),
		},
		{
			name:   "ipv4_only",
			plugin: MdnsMeshPlugin{addrMode: IPv4Only},
			expected: mustParseAddrPorts(
				"127.0.0.1:10", "2.2.2.2:10", "3.3.3.3:10",
			),
		},
		{
			name:   "prefer_ipv4",
			plugin: MdnsMeshPlugin{addrMode: PreferIPv4},
			expected: mustParseAddrPorts(
				"127.0.0.1:10", "2.2.2.2:10", "3.3.3.3:10",
				"[::1]:10", "[::2]:10", "[::3]:10",
			),
		},
		{
			name:     "filter",
			plugin:   MdnsMeshPlugin{filter: regexp.MustCompile("nothing")},
			expected: mustParseAddrPorts(),
		},
		{
			name:   "filter_include",
			plugin: MdnsMeshPlugin{filter: regexp.MustCompile(".*"), addrMode: PreferIPv6},
			expected: mustParseAddrPorts(
				"[::1]:10", "[::2]:10", "[::3]:10",
				"127.0.0.1:10", "2.2.2.2:10", "3.3.3.3:10",
			),
		},
		{
			name:   "ignore_self",
			plugin: MdnsMeshPlugin{ignoreSelf: true, addrMode: PreferIPv6},
			expected: mustParseAddrPorts(
				"[::2]:10", "[::3]:10",
				"2.2.2.2:10", "3.3.3.3:10",
			),
		},
		{
			name:   "addrs_per_host",
			plugin: MdnsMeshPlugin{addrsPerHost: 2, addrMode: PreferIPv6},
			expected: mustParseAddrPorts(
				"[::1]:10", "[::2]:10",
			),
		},
		{
			name:   "addrs_per_host_v4_only",
			plugin: MdnsMeshPlugin{addrMode: IPv4Only, addrsPerHost: 2},
			expected: mustParseAddrPorts(
				"127.0.0.1:10", "2.2.2.2:10",
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resultingHosts := tc.plugin.hostsForZeroconfServiceEntry(&entry)

			// Handle case where expected is empty
			if len(tc.expected) == 0 && len(resultingHosts) == 0 {
				return
			}

			if !reflect.DeepEqual(tc.expected, resultingHosts) {
				t.Errorf("Resulting hosts do not match expected hosts.\nExpected: %v\nGot:      %v", tc.expected, resultingHosts)
			}
		})
	}
}
