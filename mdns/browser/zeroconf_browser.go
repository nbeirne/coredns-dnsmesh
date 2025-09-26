package browser

// Requirements of this mDNS Browser:
// 1. It should have a long running zeroconf.NewResolver().Browse process running.
// 2. When a new entry is received, it should be tracked in MdnsBrowser.services.
// 3. When an entry with TTL = 0 is received, it should be removed from MdnsBrowser.services.
// 4. When an entries TTL reaches 20% of its original value, a single zeroconf.Lookup() call should happen.
// 4. When an entry's TTL reaches 20% of its original value, a single zeroconf.Lookup() call should happen.
// 5. When an entries TTL reaches 0, it should be removed from the Services() list.
// 6. The stop function should wait until all go routines are finished (especially the browseLoop and receiveEntries routines)
// 7. Every hour the Browse session should be restarted.

import (
	"context"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	TTLRefreshThreshold = 0.8
	// JitterFactor determines the random variation applied to the refresh timer
	// to prevent thundering herd problems. A value of 0.1 means +/- 10% jitter.
	JitterFactor = 0.1
	// MaxLookupTimeout is the maximum duration for a proactive mDNS lookup.
	// A lookup shouldn't take longer than this on a local network.
	MaxLookupTimeout = 15 * time.Second
)

// Main browsing interface for zeroconf.
// This service will manage zeroconf browsing sessions and it will provide
// a list of services through a cache object.
type ZeroconfBrowser struct {
	Log Logger

	domain     string
	service    string
	interfaces *[]net.Interface // subnet to search on

	zeroConfImpl ZeroconfInterface

	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	cancelBrowse context.CancelFunc
	timersMutex  sync.RWMutex
	cache        *serviceCache
	timers       map[string]*time.Timer
}

func NewZeroconfBrowser(domain, mdnsType string, interfaces *[]net.Interface) (browser *ZeroconfBrowser) {
	browser = &ZeroconfBrowser{}
	browser.service = mdnsType
	browser.domain = domain
	browser.interfaces = interfaces
	browser.Log = NoLogger{}

	browser.zeroConfImpl = ZeroconfImpl{}

	browser.cache = newServiceCache()
	browser.timers = make(map[string]*time.Timer)
	return browser
}

func (m *ZeroconfBrowser) Start() error {
	m.Log.Infof("Starting mDNS browser...")
	m.startOnce.Do(func() {
		m.wg.Add(1)
		go m.browseLoop() // browseLoop will call wg.Done() when it exits
	})
	return nil
}

func (m *ZeroconfBrowser) Stop() {
	m.stopOnce.Do(func() {
		m.Log.Infof("Stopping MdnsBrowser...")
		if cancel := m.cancelBrowse; cancel != nil {
			cancel()
		}

		// Stop all active timers
		m.timersMutex.Lock()
		for _, timer := range m.timers {
			timer.Stop()
		}
		m.timersMutex.Unlock()

		m.wg.Wait()
		m.Log.Infof("Stopped MdnsBrowser.")
	})
}

func (m *ZeroconfBrowser) Services() []*zeroconf.ServiceEntry {
	return m.cache.getServices()
}

func (m *ZeroconfBrowser) Service() string {
	return m.service
}

func (m *ZeroconfBrowser) Domain() string {
	return m.domain
}

func (m *ZeroconfBrowser) Interfaces() *[]net.Interface {
	return m.interfaces
}

func (m *ZeroconfBrowser) browseLoop() {
	outerCtx, outerCancel := context.WithCancel(context.Background())
	m.cancelBrowse = outerCancel

	entriesCh := make(chan *zeroconf.ServiceEntry, 10)

	// This goroutine handles processing entries and shutting down.
	go func() {
		defer m.wg.Done()
		m.processEntries(outerCtx, entriesCh)
	}()

	session := NewZeroconfSession(m.zeroConfImpl, m.interfaces)
	session.Browse(outerCtx, m.service, m.domain, entriesCh)
	close(entriesCh) // Close entriesCh only after the current session closes.
}

func (m *ZeroconfBrowser) processEntries(ctx context.Context, entriesCh chan *zeroconf.ServiceEntry) {
	for entry := range entriesCh {
		if entry == nil {
			continue
		}

		if entry.TTL == 0 {
			m.Log.Infof("Service '%s' announced goodbye (TTL=0), removing from cache.", entry.Instance)
			m.cache.removeEntry(entry.Instance)
			m.stopRefreshForEntry(entry)
		} else {
			// Only log the full details if the service is new.
			if m.cache.getExpiry(entry.Instance).IsZero() {
				m.Log.Infof("Discovered new service:\n    Instance: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d", entry.Instance, entry.HostName, entry.AddrIPv4, entry.AddrIPv6, entry.Port, entry.TTL)
			} else {
				m.Log.Debugf("Service updated:\n    Instance: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d", entry.Instance, entry.HostName, entry.AddrIPv4, entry.AddrIPv6, entry.Port, entry.TTL)
			}
			m.cache.addEntry(entry)
			m.scheduleRefreshForEntry(ctx, entry, entriesCh) // may write to entriesCh
		}
	}
}

func (m *ZeroconfBrowser) stopRefreshForEntry(entry *zeroconf.ServiceEntry) {
	m.timersMutex.Lock()
	defer m.timersMutex.Unlock()
	if timer, ok := m.timers[entry.Instance]; ok {
		timer.Stop()
		delete(m.timers, entry.Instance)
	}
}

func (m *ZeroconfBrowser) scheduleRefreshForEntry(ctx context.Context, entry *zeroconf.ServiceEntry, entriesCh chan<- *zeroconf.ServiceEntry) {
	// Calculate the base refresh time (e.g., 80% of TTL), plus some jitter.
	baseRefreshSeconds := float64(entry.TTL) * TTLRefreshThreshold
	jitter := (rand.Float64()*2 - 1) * JitterFactor * baseRefreshSeconds
	refreshDuration := time.Duration((baseRefreshSeconds + jitter) * float64(time.Second))

	m.Log.Debugf("Refresh scheduled for service: %v in %v", entry.Instance, refreshDuration)

	// Stop any existing timer for this service instance
	m.stopRefreshForEntry(entry)

	m.timers[entry.Instance] = time.AfterFunc(refreshDuration, func() {
		// We are about to perform a lookup. We need to know if the lookup
		// actually finds and sends a new entry to the channel. We can't
		// easily inspect the channel, so we'll check the cache.
		// We record the expiry of the *current* entry. If, after the lookup,
		// the expiry time for this service instance has not changed, it means
		// no new entry was received, and we should remove the service.
		originalExpiry := m.cache.getExpiry(entry.Instance)

		// Use a timeout for the lookup to prevent it from hanging indefinitely.
		lookupTimeout := (time.Duration(entry.TTL) * time.Second) - refreshDuration
		// Clamp the lookup timeout to a reasonable maximum (e.g., 15s).
		lookupTimeout = min(lookupTimeout, MaxLookupTimeout)
		m.Log.Debugf("TTL for %v is low, performing lookup with timeout %v", entry.Instance, lookupTimeout)
		lCtx, lCancel := context.WithTimeout(ctx, lookupTimeout)
		defer lCancel()

		session := NewZeroconfSession(m.zeroConfImpl, m.interfaces)
		err := session.Lookup(lCtx, entry.Instance, m.service, m.domain, entriesCh)

		// If the lookup failed, or if it succeeded but found no entries (i.e., the
		// cache entry was not updated with a new expiry), remove the service.
		currentExpiry := m.cache.getExpiry(entry.Instance)
		if err != nil {
			m.Log.Warningf("Lookup for service '%s' failed with error, removing from cache: %v", entry.Instance, err)
			m.stopRefreshForEntry(entry)
			localEntry := *entry
			localEntry.TTL = 0
			entriesCh <- &localEntry
		} else if !currentExpiry.After(originalExpiry) {
			m.Log.Warningf("Lookup for service '%s' succeeded but service did not respond, removing from cache.", entry.Instance)
			m.stopRefreshForEntry(entry)
			localEntry := *entry
			localEntry.TTL = 0
			entriesCh <- &localEntry
		} else {
			m.Log.Debugf("Lookup complete for %s...", entry.Instance)
		}
	})
}
