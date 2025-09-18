package browser

import (
	"context"
	"errors"
	"sort"
	"time"
	"testing"
	// "github.com/stretchr/testify/require"

	"github.com/grandcat/zeroconf"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)


func TestQueryServiceStartup(t *testing.T) {
	b := MdnsBrowser{}
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
		b.zeroConfImpl = tc.zeroconfImpl
		entriesCh := make(chan *zeroconf.ServiceEntry)
		ctx, cancel := context.WithCancel(context.Background())
		result := b.browseMdns(ctx, entriesCh)
		cancel()
		<- ctx.Done()
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

func TestMdnsBrowserDoesAddService(t *testing.T) {
	clog.D.Set()
	testCases := []struct {
		name        string
		input    []*zeroconf.ServiceEntry
		sleepBeforeExpectation time.Duration
		expected []*zeroconf.ServiceEntry
		expectedBrowseCalls int
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
		// {
		// 	name: "ttl_expired_explicit",
		// 	input: 	  []*zeroconf.ServiceEntry {
		// 		newEntry("host0", 120),
		// 		newEntry("host1", 100),
		// 		newEntry("host0", 0),
		// 	},
		// 	expected: []*zeroconf.ServiceEntry {
		// 		newEntry("host1", 100),
		// 	},
		// },

		// {
		// 	name: "ttl_at_0_does_remove",
		// 	input: 	  []*zeroconf.ServiceEntry {
		// 		newEntry("host0", 1), // refresh expected at 0.6s
		// 	},
		// 	sleepBeforeExpectation: 2 * time.Second,
		// 	expected: []*zeroconf.ServiceEntry {
		// 	},
		// 	expectedBrowseCalls: 2,
		// },

		// {
		// 	name: "ttl_at_20pct_does_refresh",
		// 	input: 	  []*zeroconf.ServiceEntry {
		// 		newEntry("host0", 6), // refresh expected at 0.6s
		// 	},
		// 	sleepBeforeExpectation: 5 * time.Second, // 80% of 6 is 4.8
		// 	expected: []*zeroconf.ServiceEntry {
		// 		newEntry("host0", 6), // should still be present
		// 	},
		// 	expectedBrowseCalls: 2,
		// },

	}

	for _, tc := range testCases {
		t.Logf("\nStarting test: %s\n", tc.name)
		fakeZeroconf := &controllableFakeZeroconf{
			entriesCh: make(chan *zeroconf.ServiceEntry),
		}
	
		browser := NewMdnsBrowser(".local", "_type", nil)
		browser.zeroConfImpl = fakeZeroconf
		browser.Start()

		for _, entry := range tc.input {
			fakeZeroconf.entriesCh <- entry
		}

		if tc.sleepBeforeExpectation > 0 {
			time.Sleep(tc.sleepBeforeExpectation)
		}

		browser.Stop()

		resultServices := browser.Services()
		sort.Sort(ByInstance(resultServices))
		sort.Sort(ByInstance(tc.expected))

		if len(resultServices) != len(tc.expected) {
			t.Errorf("Unexpected result in test %s, result is different length than expected (%d != %d)", tc.name, len(resultServices), len(tc.expected))
			continue
		}
	
		if tc.expectedBrowseCalls > 0 && fakeZeroconf.browseCalls != tc.expectedBrowseCalls {
			t.Errorf("Unexpected result in test %s, browseCalls is different than expected (%d != %d)", tc.name, fakeZeroconf.browseCalls, tc.expectedBrowseCalls)
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
	close(entries)
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

// A fake that can be controlled by the test
type controllableFakeZeroconf struct {
	entriesCh chan *zeroconf.ServiceEntry
	browseCalls int
}

func (zc *controllableFakeZeroconf) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	// Pass the browseStarted channel to the resolver
	return &controllableFakeResolver{entriesCh: zc.entriesCh, browseCalls: &zc.browseCalls}, nil
}

type controllableFakeResolver struct {
	entriesCh chan *zeroconf.ServiceEntry
	browseCalls *int
}

func (r *controllableFakeResolver) Browse(ctx context.Context, service, domain string, entries chan<- *zeroconf.ServiceEntry) error {
	*r.browseCalls = *r.browseCalls + 1

	// This fake browse blocks and waits for the test to send entries or the context to be canceled.
	for {
		select {
		case <-ctx.Done():
			close(entries)
			return nil
		case entry := <-r.entriesCh:
			entries <- entry
		}
	}
}

// sorting

type ByInstance []*zeroconf.ServiceEntry
func (a ByInstance) Len() int           { return len(a) }
func (a ByInstance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByInstance) Less(i, j int) bool { return a[i].Instance < a[j].Instance }

func newEntry(instanceName string, ttl uint32) (entry *zeroconf.ServiceEntry) {
	e := zeroconf.ServiceEntry { ServiceRecord: zeroconf.ServiceRecord { Instance: instanceName }, TTL: ttl }
	entry = &e
	return entry
}
