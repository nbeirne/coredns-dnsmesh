
package mdns 

import (
	"slices"
	"testing"
	"regexp"
	"net"
	"net/netip"

	"github.com/celebdor/zeroconf"
)


func TestHostFiltering(t *testing.T) {

	entry := zeroconf.ServiceEntry {
		ServiceRecord: zeroconf.ServiceRecord { Instance: "test_instance_name" },
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

	testCases := [] struct {
		name     string
		plugin   MdnsMeshPlugin
		expected []netip.AddrPort
	}{
		{
			name: "defaults",
			plugin: MdnsMeshPlugin {},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("[::1]:10"),
				netip.MustParseAddrPort("[::2]:10"),
				netip.MustParseAddrPort("[::3]:10"),
				netip.MustParseAddrPort("127.0.0.1:10"),
				netip.MustParseAddrPort("2.2.2.2:10"),
				netip.MustParseAddrPort("3.3.3.3:10"),
			},
		},
		{
			name: "ipv6_only",
			plugin: MdnsMeshPlugin {
				addrMode: IPv6Only,
			},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("[::1]:10"),
				netip.MustParseAddrPort("[::2]:10"),
				netip.MustParseAddrPort("[::3]:10"),
			},
		},
		{
			name: "ipv4_only",
			plugin: MdnsMeshPlugin {
				addrMode: IPv4Only,
			},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("127.0.0.1:10"),
				netip.MustParseAddrPort("2.2.2.2:10"),
				netip.MustParseAddrPort("3.3.3.3:10"),
			},
		},
		{
			name: "prefer_ipv4",
			plugin: MdnsMeshPlugin {
				addrMode: PreferIPv4,
			},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("127.0.0.1:10"),
				netip.MustParseAddrPort("2.2.2.2:10"),
				netip.MustParseAddrPort("3.3.3.3:10"),
				netip.MustParseAddrPort("[::1]:10"),
				netip.MustParseAddrPort("[::2]:10"),
				netip.MustParseAddrPort("[::3]:10"),
			},
		},
		{
			name: "filter",
			plugin: MdnsMeshPlugin {
				filter: regexp.MustCompile("nothing"),
			},
			expected: []netip.AddrPort {
			},
		},
		{
			name: "filter_include",
			plugin: MdnsMeshPlugin {
				filter: regexp.MustCompile(".*"),
			},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("[::1]:10"),
				netip.MustParseAddrPort("[::2]:10"),
				netip.MustParseAddrPort("[::3]:10"),
				netip.MustParseAddrPort("127.0.0.1:10"),
				netip.MustParseAddrPort("2.2.2.2:10"),
				netip.MustParseAddrPort("3.3.3.3:10"),
			},
		},
		{
			name: "ignore_self",
			plugin: MdnsMeshPlugin {
				ignoreSelf: true,
			},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("[::2]:10"),
				netip.MustParseAddrPort("[::3]:10"),
				netip.MustParseAddrPort("2.2.2.2:10"),
				netip.MustParseAddrPort("3.3.3.3:10"),
			},
		},
		{
			name: "addrs_per_host",
			plugin: MdnsMeshPlugin {
				addrsPerHost: 2,
			},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("[::1]:10"),
				netip.MustParseAddrPort("[::2]:10"),
			},
		},
		{
			name: "addrs_per_host_v4_only",
			plugin: MdnsMeshPlugin {
				addrMode: IPv4Only,
				addrsPerHost: 2,
			},
			expected: []netip.AddrPort {
				netip.MustParseAddrPort("127.0.0.1:10"),
				netip.MustParseAddrPort("2.2.2.2:10"),
			},
		},
	}

	for _, tc := range testCases {
		resultingHosts := tc.plugin.hostsForZeroconfServiceEntry(&entry)

		if !slices.Equal(tc.expected, resultingHosts) {
			t.Errorf("test: %s: resulting hosts do not match expected hosts.\nExpected: %v\nGot:      %v", tc.name, tc.expected, resultingHosts)
		}
	}
}
