package mdns

import (
	"context"
	"errors"
	"net"
	//"sync"
	"sort"
	"testing"

	"github.com/celebdor/zeroconf"
	//"github.com/coredns/coredns/request"
	//"github.com/miekg/dns"
)


func TestQueryServiceStartup(t *testing.T) {
	testCases := []struct {
		tcase         string
		expectedError string
		zeroconfImpl  ZeroconfInterface
	}{
		{"queryService succeeds", "", fakeZeroconf{}},
		{"NewResolver fails", "test resolver error", failZeroconf{}},
		{"Browse fails", "test browse error", browseFailZeroconf{}},
	}
	for _, tc := range testCases {
		entriesCh := make(chan *zeroconf.ServiceEntry)
		result := queryService("test", entriesCh, []net.Interface{}, tc.zeroconfImpl)
		if tc.expectedError == "" {
			if result != nil {
				t.Errorf("Unexpected failure in %v: %v", tc.tcase, result)
			}
		} else {
			if result.Error() != tc.expectedError {
				t.Errorf("Unexpected result in %v: %v", tc.tcase, result)
			}
		}
	}
}

func newEntry(instanceName string, ttl uint32) (entry *zeroconf.ServiceEntry) {
	e := zeroconf.ServiceEntry { ServiceRecord: zeroconf.ServiceRecord { Instance: instanceName }, TTL: ttl }
	entry = &e
	return entry
}

func TestDnsBrowser(t *testing.T) {

	testCases := [] struct {
		name        string
		input    []*zeroconf.ServiceEntry
		expected []*zeroconf.ServiceEntry
	}{
		{
			name: "basic",
			input: 	  []*zeroconf.ServiceEntry { 
				newEntry("host0", 120),
			},
			expected: []*zeroconf.ServiceEntry { 
				newEntry("host0", 120),
			},
		},
		{
			name: "two_hosts",
			input: 	  []*zeroconf.ServiceEntry { 
				newEntry("host0", 120),
				newEntry("host1", 100),
			},
			expected: []*zeroconf.ServiceEntry { 
				newEntry("host0", 120),
				newEntry("host1", 100),
			},
		},
		{
			name: "updated_host",
			input: 	  []*zeroconf.ServiceEntry { 
				newEntry("host0", 120),
				newEntry("host0", 100),
				newEntry("host0", 90),
			},
			expected: []*zeroconf.ServiceEntry { 
				newEntry("host0", 90),
			},
		},
		//{
		//	name: "ttl_expired",
		//	input: 	  []*zeroconf.ServiceEntry { 
		//		newEntry("host0", 120),
		//		newEntry("host1", 100),
		//		newEntry("host0", 0),
		//	},
		//	expected: []*zeroconf.ServiceEntry { 
		//		newEntry("host1", 100),
		//	},
		//},

		// TODO: remove entry with 0 TTL
	}

	for _, tc := range testCases {
		browser := NewMdnsBrowser("_type", nil)
		browser.zeroConfImpl = fakeZeroconf{tc.input}

		browser.browseMdns()

		resultServices := browser.Services()

		sort.Sort(ByInstance(resultServices))
		sort.Sort(ByInstance(tc.expected))

		if len(resultServices) != len(tc.expected) {
			t.Errorf("Unexpected result in test %s, result is different length than expected (%d != %d)", tc.name, len(resultServices), len(tc.expected))
		}

		for idx := range resultServices {
			if resultServices[idx].HostName != tc.expected[idx].HostName {
				t.Errorf("Unexpected result in test %s, hostname at %d is different expected (%s != %s)", tc.name, idx, resultServices[idx].HostName, tc.expected[idx].HostName)
			}
			if resultServices[idx].TTL != tc.expected[idx].TTL {
				t.Errorf("Unexpected result in test %s, ttl at %d is different expected (%d != %d)", tc.name, idx, resultServices[idx].TTL, tc.expected[idx].TTL)
			}
		}
	}
}


// setup fakes
type fakeZeroconf struct{
	entries []*zeroconf.ServiceEntry
}

func (zc fakeZeroconf) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	return fakeResolver{zc.entries}, nil
}

type failZeroconf struct{
}

func (failZeroconf) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	return nil, errors.New("test resolver error")
}

type fakeResolver struct{
	entries []*zeroconf.ServiceEntry
}

func (r fakeResolver) Browse(context context.Context, service, domain string, entries chan<- *zeroconf.ServiceEntry) error {
	for _, entry := range r.entries {
		entries <- entry
	}
	return nil
}

type browseFailZeroconf struct{}

func (browseFailZeroconf) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	return failResolver{}, nil
}

type failResolver struct{}

func (failResolver) Browse(context context.Context, service, domain string, entries chan<- *zeroconf.ServiceEntry) error {
	return errors.New("test browse error")
}


// sorting

type ByInstance []*zeroconf.ServiceEntry
func (a ByInstance) Len() int           { return len(a) }
func (a ByInstance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByInstance) Less(i, j int) bool { return a[i].Instance < a[j].Instance }


