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
	"net"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	TTLRefreshThreshold = 0.8
	// MaxLookupTimeout is the maximum duration for a proactive mDNS lookup.
	// A lookup shouldn't take longer than this on a local network.
	MaxLookupTimeout = 15 * time.Second
)

var (
	// JitterFactor determines the random variation applied to the refresh timer
	// to prevent thundering herd problems. A value of 0.1 means +/- 10% jitter.
	JitterFactor = 0.1
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

	entriesCh chan *zeroconf.ServiceEntry
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	cancelBrowse context.CancelFunc // Cancels the main browse loop and all derived contexts
	cache        *serviceCache
	refresher    *ServiceRefresher
}

func NewZeroconfBrowser(domain, mdnsType string, interfaces *[]net.Interface) (browser *ZeroconfBrowser) {
	browser = &ZeroconfBrowser{}
	browser.service = mdnsType
	browser.domain = domain
	browser.interfaces = interfaces
	browser.Log = NoLogger{}

	browser.zeroConfImpl = ZeroconfImpl{}

	browser.cache = newServiceCache()
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

		m.Log.Debugf("Stopping refreshes...")
		m.refresher.StopAll()

		m.Log.Debugf("Waiting for all routines to stop...")
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

// ForceRefresh triggers a non-blocking, one-shot mDNS browse operation to quickly
// rediscover services on the network. This is useful to call reactively when an
// operation like a DNS query fails, as it can refresh the cache with up-to-date
// service information without waiting for the next TTL-based refresh.
func (m *ZeroconfBrowser) ForceRefresh(ctx context.Context) {
	m.Log.Infof("Force-refresh triggered. Performing a one-shot browse for '%s'.", m.service)
	session := NewZeroconfSession(m.zeroConfImpl, m.interfaces)
	_ = session.Browse(ctx, m.service, m.domain, m.entriesCh)
}

func (m *ZeroconfBrowser) browseLoop() {
	outerCtx, outerCancel := context.WithCancel(context.Background())
	m.cancelBrowse = outerCancel

	m.entriesCh = make(chan *zeroconf.ServiceEntry, 10)
	session := NewZeroconfSession(m.zeroConfImpl, m.interfaces)

	m.refresher = newServiceRefresher(m.service, m.domain, session, m.cache, m.entriesCh, m.removeService)
	m.refresher.Log = m.Log

	// This goroutine handles processing entries and shutting down.
	go func() {
		defer m.wg.Done()
		m.processEntries(outerCtx, m.entriesCh)
	}()

	// This call blocks until the context is canceled.
	_ = session.Browse(outerCtx, m.service, m.domain, m.entriesCh)
	close(m.entriesCh)
}

func (m *ZeroconfBrowser) processEntries(ctx context.Context, entriesCh chan *zeroconf.ServiceEntry) {
	for entry := range entriesCh {
		if entry == nil {
			continue
		}

		if entry.TTL == 0 {
			m.Log.Infof("Service '%s' announced goodbye (TTL=0), removing from cache.", entry.Instance)
			m.removeService(entry)
		} else {
			// Only log the full details if the service is new.
			if m.cache.getExpiry(entry.Instance).IsZero() {
				m.Log.Infof("Discovered new service:\n    Instance: %s\n    Service: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d", entry.Instance, entry.Service, entry.HostName, entry.AddrIPv4, entry.AddrIPv6, entry.Port, entry.TTL)
			} else {
				m.Log.Debugf("Service updated:\n    Instance: %s\n    Service: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d", entry.Instance, entry.Service, entry.HostName, entry.AddrIPv4, entry.AddrIPv6, entry.Port, entry.TTL)
			}
			m.cache.addEntry(entry)
			m.refresher.Refresh(ctx, entry)
		}
	}
}

func (m *ZeroconfBrowser) removeService(entry *zeroconf.ServiceEntry) {
	m.cache.removeEntry(entry.Instance)
	// No need to call refresher.Stop() here as the timer will fire and do nothing,
	// and the next time a service with this name appears, Refresh() will overwrite the timer.
}
