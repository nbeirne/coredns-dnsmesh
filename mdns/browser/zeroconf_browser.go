package browser

// Requirements of this mDNS Browser:
// 1. It should have a long running zeroconf.NewResolver().Browse process running.
// 2. When a new entry is received, it should be tracked in MdnsBrowser.services.
// 3. When an entry with TTL = 0 is received, it should be removed from MdnsBrowser.services.
// 4. When an entries TTL reaches 20% of its original value, a single zeroconf.Lookup() call should happen.
// 5. When an entries TTL reaches 0, it should be removed from the Services() list.
// 6. The stop function should wait until all go routines are finished (especially the browseLoop and receiveEntries routines)
// 7. Every hour the Browse session should be restarted.

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	TTLRefreshThreshold = 0.8
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
			m.handleRemovedService(entry)
		} else {
			m.handleDiscoveredService(ctx, entry, entriesCh) // may write to entriesCh
		}
	}
}

func (m *ZeroconfBrowser) handleRemovedService(entry *zeroconf.ServiceEntry) {
	m.Log.Warningf("Service removed/expired: %s", entry.Instance)
	m.cache.removeEntry(entry.Instance)

	m.timersMutex.Lock()
	defer m.timersMutex.Unlock()
	if timer, ok := m.timers[entry.Instance]; ok {
		timer.Stop()
		delete(m.timers, entry.Instance)
	}
}

func (m *ZeroconfBrowser) handleDiscoveredService(ctx context.Context, entry *zeroconf.ServiceEntry, entriesCh chan<- *zeroconf.ServiceEntry) {
	refreshDuration := time.Duration(entry.TTL) * time.Duration(float64(time.Second)*TTLRefreshThreshold)

	m.Log.Infof("Discovered service\n    Instance: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d (refresh in %v)", entry.Instance, entry.HostName, entry.AddrIPv4, entry.AddrIPv6, entry.Port, entry.TTL, refreshDuration)

	// Add or update the entry in the cache
	m.cache.addEntry(entry)

	// Stop any existing timer for this service instance
	m.timersMutex.Lock()
	defer m.timersMutex.Unlock()
	if timer, ok := m.timers[entry.Instance]; ok {
		timer.Stop()
	}

	// Create a new timer to trigger a lookup for this service.
	m.timers[entry.Instance] = time.AfterFunc(refreshDuration, func() {
		// Use a timeout for the lookup to prevent it from hanging indefinitely.
		lookupTimeout := (time.Duration(entry.TTL) * time.Second) - refreshDuration
		m.Log.Infof("TTL for %v is low, performing lookup lasting %v", entry.Instance, lookupTimeout)
		lCtx, lCancel := context.WithTimeout(ctx, lookupTimeout)
		defer lCancel()

		session := NewZeroconfSession(m.zeroConfImpl, m.interfaces)
		err := session.Lookup(lCtx, entry.Instance, m.service, m.domain, entriesCh)

		// We need a new resolver for a one-off lookup.
		if err != nil {
			m.Log.Warningf("Failed to lookup service %s, removing from cache: %v", entry.Instance, err)
			m.handleRemovedService(entry)
		} else {
			m.Log.Debugf("Lookup complete for %s...", entry.Instance)
		}
	})
}
