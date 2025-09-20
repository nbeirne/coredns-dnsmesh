package browser

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	// "github.com/stretchr/testify/require"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/grandcat/zeroconf"
)

func TestZeroconfBrowserDoesAddService(t *testing.T) {
	clog.D.Set()
	testCases := []struct {
		name                   string
		input                  []*zeroconf.ServiceEntry
		sleepBeforeExpectation time.Duration
		expected               []*zeroconf.ServiceEntry
		expectedBrowseCalls    int
	}{
		{
			name: "basic",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
			},
			expected: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
			},
		},
		{
			name: "two_hosts",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
				newEntry("host1", 100),
			},
			expected: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
				newEntry("host1", 100),
			},
		},
		{
			name: "updated_host",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
				newEntry("host0", 100),
				newEntry("host0", 90),
			},
			expected: []*zeroconf.ServiceEntry{
				newEntry("host0", 90),
			},
		},
		{
			name: "ttl_expired_explicit",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
				newEntry("host1", 100),
				newEntry("host2", 40),
				newEntry("host0", 0),
			},
			expected: []*zeroconf.ServiceEntry{
				newEntry("host1", 100),
				newEntry("host2", 40),
			},
		},
		{
			name: "many_hosts_added_removed",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
				newEntry("host1", 100),
				newEntry("host2", 100),
				newEntry("host3", 100),
				newEntry("host4", 100),
				newEntry("host5", 100),
				newEntry("host6", 100),
				newEntry("host0", 120),
				newEntry("host1", 100),
				newEntry("host2", 100),
				newEntry("host3", 100),
				newEntry("host4", 100),
				newEntry("host5", 100),
				newEntry("host6", 100),
				newEntry("host0", 120),
				newEntry("host1", 120),
				newEntry("host2", 0),
				newEntry("host3", 0),
				newEntry("host4", 0),
				newEntry("host5", 100),
				newEntry("host6", 100),
			},
			expected: []*zeroconf.ServiceEntry{
				newEntry("host0", 120),
				newEntry("host1", 120),
				newEntry("host5", 100),
				newEntry("host6", 100),
			},
		},
	}

	for _, tc := range testCases {
		t.Logf("\nStarting test: %s\n", tc.name)
		fakeZeroconf := &controllableFakeZeroconf{
			browseEntriesCh: make(chan *zeroconf.ServiceEntry),
			lookupEntriesCh: make(chan *zeroconf.ServiceEntry),
		}

		browser := NewZeroconfBrowser(".local", "_type", nil)
		browser.zeroConfImpl = fakeZeroconf
		browser.Start()

		for _, entry := range tc.input {
			fakeZeroconf.browseEntriesCh <- entry
		}

		if tc.sleepBeforeExpectation > 0 {
			time.Sleep(tc.sleepBeforeExpectation)
		}

		browser.Stop()

		resultServices := browser.Services()
		sort.Sort(ByInstance(resultServices))
		sort.Sort(ByInstance(tc.expected))

		if tc.expectedBrowseCalls > 0 && fakeZeroconf.browseCalls != tc.expectedBrowseCalls {
			t.Errorf("Unexpected result in test %s, browseCalls is different than expected (%d != %d)", tc.name, fakeZeroconf.browseCalls, tc.expectedBrowseCalls)
		}

		if len(resultServices) != len(tc.expected) {
			t.Errorf("Unexpected result in test %s, result is different length than expected (%d != %d)", tc.name, len(resultServices), len(tc.expected))
			continue
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

func TestTTL(t *testing.T) {
	clog.D.Set()
	testCases := []struct {
		name                   string
		input                  []*zeroconf.ServiceEntry
		sleepBeforeExpectation time.Duration
		expected               []*zeroconf.ServiceEntry
		expectedBrowseCalls    int
		expectedLookupCalls    map[string]int
	}{
		{
			name: "pre_ttl_does_not_refresh",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 2),
			},
			sleepBeforeExpectation: 1 * time.Second,
			expected: []*zeroconf.ServiceEntry{
				newEntry("host0", 2),
			},
			expectedBrowseCalls: 1,
			expectedLookupCalls: map[string]int{
				"host0": 0,
			},
		},

		{
			name: "ttl_at_20pct_does_refresh",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 6), // refresh expected at 4.8s
			},
			sleepBeforeExpectation: 5 * time.Second, // 80% of 6 is 4.8
			expected: []*zeroconf.ServiceEntry{
				newEntry("host0", 6), // should still be present
			},
			expectedBrowseCalls: 1,
			expectedLookupCalls: map[string]int{
				"host0": 1,
			},
		},

		{
			name: "ttl_at_0_does_remove",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 1), // refresh expected at 0.6s
			},
			sleepBeforeExpectation: 2 * time.Second,
			expected:               []*zeroconf.ServiceEntry{},
			expectedBrowseCalls:    1,
			expectedLookupCalls: map[string]int{
				"host0": 1,
			},
		},

		{
			name: "ttl_expiring_does_not_spam",
			input: []*zeroconf.ServiceEntry{
				newEntry("host0", 10), // refresh expected at 8s
				newEntry("host1", 6),  // refresh expected at 8s
			},
			sleepBeforeExpectation: 20 * time.Second, // 80% of 6 is 4.8
			expected:               []*zeroconf.ServiceEntry{},
			expectedBrowseCalls:    1,
			expectedLookupCalls: map[string]int{
				"host0": 1,
				"host1": 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Logf("\nStarting test: %s\n", tc.name)
		fakeZeroconf := controllableFakeZeroconf{
			browseEntriesCh: make(chan *zeroconf.ServiceEntry),
			lookupEntriesCh: make(chan *zeroconf.ServiceEntry),
			lookupCalls:     make(map[string]int),
		}

		browser := NewZeroconfBrowser(".local", "_type", nil)
		browser.zeroConfImpl = &fakeZeroconf
		browser.Start()

		for _, entry := range tc.input {
			fakeZeroconf.browseEntriesCh <- entry
		}

		if tc.sleepBeforeExpectation > 0 {
			time.Sleep(tc.sleepBeforeExpectation)
		}

		browser.Stop()

		resultServices := browser.Services()
		sort.Sort(ByInstance(resultServices))
		sort.Sort(ByInstance(tc.expected))

		if tc.expectedBrowseCalls > 0 && fakeZeroconf.browseCalls != tc.expectedBrowseCalls {
			t.Errorf("Unexpected result in test %s, browseCalls is different than expected (%d != %d)", tc.name, fakeZeroconf.browseCalls, tc.expectedBrowseCalls)
		}

		for name, expectedLookups := range tc.expectedLookupCalls {
			if fakeZeroconf.lookupCalls[name] != expectedLookups {
				t.Errorf("Unexpected result in test %s, lookupCalls for %s is different than expected (%d != %d)", tc.name, name, fakeZeroconf.lookupCalls[name], expectedLookups)
			}
		}

		if len(resultServices) != len(tc.expected) {
			t.Errorf("Unexpected result in test %s, result is different length than expected (%d != %d)", tc.name, len(resultServices), len(tc.expected))
			continue
		}

		for idx := range resultServices {
			name := tc.expected[idx].Instance
			if resultServices[idx].HostName != tc.expected[idx].HostName {
				t.Errorf("Unexpected result in test %s, hostname for %s is different expected (%s != %s)", tc.name, name, resultServices[idx].HostName, tc.expected[idx].HostName)
			}
			if resultServices[idx].TTL != tc.expected[idx].TTL {
				t.Errorf("Unexpected result in test %s, ttl at %s is different expected (%d != %d)", tc.name, name, resultServices[idx].TTL, tc.expected[idx].TTL)
			}

			if expLookup, ok := tc.expectedLookupCalls[name]; ok && fakeZeroconf.lookupCalls[name] != expLookup {
				t.Errorf("Unexpected result in test %s, lookupCalls for %s is different than expected (%d != %d)", tc.name, name, fakeZeroconf.lookupCalls[name], expLookup)
			}
		}
	}

}

// func _TestMdnsBrowser_Lifecycle(t *testing.T) {
// 	clog.D.Set()

// 	// This fake allows the test to control when mDNS entries are "discovered".
// 	fakeZeroconf := &controllableFakeZeroconf{
// 		entriesCh: make(chan *zeroconf.ServiceEntry),
// 		browseStarted: make(chan struct{}, 10), // Buffered to avoid blocking
// 	}

// 	browser := NewMdnsBrowser(".local", "_type", nil)
// 	browser.zeroConfImpl = fakeZeroconf

// 	// Start the browser. This should start the long-running browse loop.
// 	browser.Start()
// 	defer browser.Stop()

// 	// Helper to check services.
// 	assertNumServices := func(expected int, msg string) {
// 		// Poll because processing is asynchronous
// 		require.Eventually(t, func() bool {
// 			return len(browser.Services()) == expected
// 		}, 2*time.Second, 50*time.Millisecond, "Assertion failed: "+msg)
// 	}

// 	// Wait for the first browse process to start.
// 	<-fakeZeroconf.browseStarted

// 	// --- Test Requirement 2: New entry is tracked ---
// 	t.Log("Testing: New entry is tracked")
// 	entry1 := newEntry("host1", 10) // 10s TTL
// 	fakeZeroconf.entriesCh <- entry1
// 	assertNumServices(1, "Should track 1 service after adding host1")
// 	t.Log("... PASSED")

// 	// --- Test Requirement 4: Browse is restarted on TTL threshold ---
// 	// The refresh should happen at 80% of TTL, which is 8 seconds for a 10s TTL.
// 	// We wait for the browse to be restarted.
// 	t.Log("Testing: Browse is restarted when TTL reaches 20%")
// 	select {
// 	case <-fakeZeroconf.browseStarted:
// 		t.Log("... PASSED: Browse was restarted as expected.")
// 	case <-time.After(9 * time.Second): // A bit more than 8s to be safe
// 		t.Fatal("FAIL: Browse was not restarted within the expected time frame.")
// 	}

// 	// After restart, the service should still be present until it expires.
// 	assertNumServices(1, "Service should still be present after refresh")

// 	// --- Test Requirement 3: Entry with TTL=0 is removed ---
// 	t.Log("Testing: Entry with TTL=0 is removed")
// 	entry2 := newEntry("host2", 120)
// 	fakeZeroconf.entriesCh <- entry2
// 	assertNumServices(2, "Should track 2 services after adding host2")

// 	goodbyeEntry := newEntry("host2", 0) // TTL 0 means "goodbye"
// 	fakeZeroconf.entriesCh <- goodbyeEntry
// 	assertNumServices(1, "Should have 1 service after host2 says goodbye")
// 	t.Log("... PASSED")

// 	// --- Test Requirement 5: Entry is removed when its TTL expires ---
// 	// The first entry for "host1" had a 10s TTL. We've already waited ~8s.
// 	// Waiting a bit longer should cause it to expire.
// 	t.Log("Testing: Entry is removed after TTL expires")

// 	// The expiry check is triggered by a new entry. Let's send one.
// 	// We wait > 2 more seconds to ensure the original 10s TTL is up.
// 	time.Sleep(3 * time.Second)
// 	fakeZeroconf.entriesCh <- newEntry("trigger-cleanup", 120)

// 	// Now only "trigger-cleanup" should be left.
// 	assertNumServices(1, "Should have 1 service after host1's TTL expired")
// 	services := browser.Services()
// 	if len(services) == 1 && services[0].Instance != "trigger-cleanup" {
// 		t.Errorf("Incorrect service remaining, expected 'trigger-cleanup', got '%s'", services[0].Instance)
// 	}
// 	t.Log("... PASSED")

// 	// --- Test Requirement 1: Long-running process ---
// 	// The fact that all previous steps worked, especially the timed refresh,
// 	// implicitly confirms a long-running process is managed correctly.
// 	t.Log("Testing: Long-running process is active")
// 	// Stop() now blocks until all goroutines are finished.
// 	browser.Stop()
// 	// If Stop() returns successfully without a timeout, it means the cleanup
// 	// worked as expected.
// 	select {
// 	case <-time.After(1 * time.Second):
// 		t.Log("... PASSED: Stop() appears to have terminated the browse loop.")
// 	}
// }

// A fake that can be controlled by the test
type controllableFakeZeroconf struct {
	browseEntriesCh chan *zeroconf.ServiceEntry
	lookupEntriesCh chan *zeroconf.ServiceEntry

	browseCalls int
	lookupCalls map[string]int

	mutex sync.Mutex
}

func (zc *controllableFakeZeroconf) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	// Pass the browseStarted channel to the resolver
	return &controllableFakeResolver{parent: zc}, nil
}

type controllableFakeResolver struct {
	parent *controllableFakeZeroconf
}

func (r *controllableFakeResolver) Browse(ctx context.Context, service, domain string, entries chan<- *zeroconf.ServiceEntry) error {
	r.parent.mutex.Lock()
	r.parent.browseCalls++
	r.parent.mutex.Unlock()

	// This fake browse blocks and waits for the test to send entries or the context to be canceled.
	for {
		select {
		case <-ctx.Done():
			close(entries)
			return nil
		case entry := <-r.parent.browseEntriesCh:
			entries <- entry
		}
	}
}

func (r *controllableFakeResolver) Lookup(ctx context.Context, instance, service, domain string, entries chan<- *zeroconf.ServiceEntry) error {
	r.parent.mutex.Lock()
	r.parent.lookupCalls[instance]++
	r.parent.mutex.Unlock()

	// This fake lookup blocks and waits for the test to send entries or the context to be canceled.
	for {
		select {
		case <-ctx.Done():
			close(entries)
			return nil
		case entry := <-r.parent.lookupEntriesCh:
			entries <- entry
		}
	}

	return nil
}

// sorting

type ByInstance []*zeroconf.ServiceEntry

func (a ByInstance) Len() int           { return len(a) }
func (a ByInstance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByInstance) Less(i, j int) bool { return a[i].Instance < a[j].Instance }

func newEntry(instanceName string, ttl uint32) (entry *zeroconf.ServiceEntry) {
	e := zeroconf.ServiceEntry{ServiceRecord: zeroconf.ServiceRecord{Instance: instanceName}, TTL: ttl}
	entry = &e
	return entry
}
