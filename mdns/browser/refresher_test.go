package browser

import (
	"context"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
)

func TestServiceRefresher(t *testing.T) {
	// For testing, we'll set the jitter to 0 to have predictable timer durations.
	originalJitterFactor := JitterFactor
	JitterFactor = 0
	defer func() { JitterFactor = originalJitterFactor }()

	testCases := []struct {
		name                string
		lookupShouldError   bool
		lookupUpdatesExpiry bool
		expectBrowseCall    bool
	}{
		{
			name:                "Successful Lookup Refreshes Entry",
			lookupShouldError:   false,
			lookupUpdatesExpiry: true,
			expectBrowseCall:    false,
		},
		{
			name:                "Failed Lookup Triggers Browse Fallback",
			lookupShouldError:   true,
			lookupUpdatesExpiry: false,
			expectBrowseCall:    true,
		},
		{
			name:                "Lookup With No Update Triggers Browse Fallback",
			lookupShouldError:   false,
			lookupUpdatesExpiry: false,
			expectBrowseCall:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Setup ---
			logger := NewTestLogger(t)
			mockCache := newServiceCache()
			entriesCh := make(chan *zeroconf.ServiceEntry, 1)

			// Use the existing controllable fake from zeroconf_browser_test.go
			fakeZeroconf := &controllableFakeZeroconf{
				logger:      logger,
				lookupCalls: make(map[string]int),
			}
			fakeZeroconf.lookupShouldError = tc.lookupShouldError
			session := NewZeroconfSession(fakeZeroconf, nil)

			// The refresher doesn't use the onRemove function directly, so we can pass nil.
			refresher := newServiceRefresher("_test._tcp", "local", session, mockCache, entriesCh, nil)
			refresher.Log = logger

			// Create a test entry with a very short TTL to make the test run quickly.
			// A 1s TTL with a 0.8 threshold means the refresh will trigger after 800ms.
			entry := &zeroconf.ServiceEntry{ServiceRecord: zeroconf.ServiceRecord{Instance: "test-instance", Service: "_test._tcp"}, TTL: 1}

			// Set the initial state of the cache and mock session.
			initialExpiry := time.Now().Add(time.Duration(entry.TTL) * time.Second)
			mockCache.setExpiry(entry, initialExpiry)

			// --- Action ---
			// Start the refresh process. This will set a timer.
			refresher.Refresh(context.Background(), entry)

			// Wait for the timer to fire. 800ms is the threshold, so we wait a bit longer.
			time.Sleep(1000 * time.Millisecond)

			// After the lookup is supposed to have happened, we configure the cache's
			// next response to simulate whether the lookup was successful in updating the entry.
			if tc.lookupUpdatesExpiry {
				mockCache.setExpiry(entry, initialExpiry.Add(time.Duration(entry.TTL)*time.Second))
			}

			// The check for expiry and the potential fallback browse happen inside the timer's
			// goroutine. We need to wait a moment for that logic to complete.
			time.Sleep(50 * time.Millisecond)

			// --- Assertions ---
			if fakeZeroconf.lookupCalls[entry.Instance] != 1 {
				t.Errorf("Expected Lookup() to be called 1 time, but was called %d times", fakeZeroconf.lookupCalls[entry.Instance])
			}

			expectedBrowseCalls := 0
			if tc.expectBrowseCall {
				expectedBrowseCalls = 1
			}
			if fakeZeroconf.browseCalls != expectedBrowseCalls {
				t.Errorf("Expected Browse() to be called %d times, but was called %d times", expectedBrowseCalls, fakeZeroconf.browseCalls)
			}

			refresher.StopAll()
		})
	}
}

func (m *serviceCache) setExpiry(entry *zeroconf.ServiceEntry, expiry time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if trackedSvc, ok := (*m.services)[entry.Instance]; ok {
		trackedSvc.expiry = expiry
		(*m.services)[entry.Instance] = trackedSvc
	} else {
		(*m.services)[entry.Instance] = &trackedService{
			entry:       entry,
			originalTTL: time.Duration(entry.TTL) * time.Second,
			expiry:      expiry,
		}
	}
}
