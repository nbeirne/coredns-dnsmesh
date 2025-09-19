package browser

// Requirements of this mDNS Browser:
// 1. It should have a long running zeroconf..NewResolver().Browse process running.
// 2. When a new entry is received, it should be tracked in MdnsBrowser.services.
// 3. When an entry with TTL = 0 is received, it should be removed from MdnsBrowser.services.
// 4. When an entries TTL reaches 20% of its original value, the Browse() process should be canceled and restarted.
// 5. When an entries TTL reaches 0, it should be removed from MdnsBrowser.services.
// 6. The stop function should wait until all go routines are finished (especially the browseLoop and receiveEntries routines)

import (
	"context"
	"net"
	"sync"

	"github.com/grandcat/zeroconf"
)

const (
	TTLRefreshThreshold = 0.8
)

// Main browsing interface for zeroconf.
// This service will manage zeroconf browsing sessions and it will provide
// a list of services through a cache object.
type ZeroconfBrowser struct {
	domain     string
	mdnsType   string
	interfaces *[]net.Interface // subnet to search on

	zeroConfImpl ZeroconfInterface

	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	cancelBrowse context.CancelFunc
	cache        *serviceCache
}

func NewZeroconfBrowser(domain, mdnsType string, interfaces *[]net.Interface) (browser *ZeroconfBrowser) {
	browser = &ZeroconfBrowser{}
	browser.mdnsType = mdnsType
	browser.domain = domain
	browser.interfaces = interfaces

	browser.zeroConfImpl = ZeroconfImpl{}

	browser.cache = newServiceCache()
	return browser
}

func (m *ZeroconfBrowser) Start() {
	log.Info("Starting mDNS browser...")
	m.startOnce.Do(func() {
		m.wg.Add(1)
		go m.browseLoop() // browseLoop will call wg.Done() when it exits
	})
}

func (m *ZeroconfBrowser) Stop() {
	m.stopOnce.Do(func() {
		log.Info("Stopping MdnsBrowser...")
		if cancel := m.cancelBrowse; cancel != nil {
			cancel()
		}

		m.wg.Wait()
		log.Info("Stopped MdnsBrowser.")
	})
}

func (m *ZeroconfBrowser) Services() []*zeroconf.ServiceEntry {
	return m.cache.getServices()
}

func (m *ZeroconfBrowser) browseLoop() {
	log.Debug("Start browseLoop....")
	defer log.Debug("Finish browseLoop....")

	outerCtx, outerCancel := context.WithCancel(context.Background())
	m.cancelBrowse = outerCancel

	entriesCh := make(chan *zeroconf.ServiceEntry, 10)
	lookupTriggerCh := make(chan *zeroconf.ServiceEntry, 10)

	// This goroutine handles processing entries and shutting down.
	go func() {
		defer m.wg.Done()
		m.handleDiscoveredService(entriesCh, lookupTriggerCh)
	}()

	session := NewZeroconfSession(m.zeroConfImpl, m.interfaces, m.mdnsType, m.domain)
	session.Run(outerCtx, entriesCh)
	close(entriesCh) // Close fanInCh only after the current session closes.
}

func (m *ZeroconfBrowser) handleDiscoveredService(entriesCh <-chan *zeroconf.ServiceEntry, lookupTriggerCh chan<- *zeroconf.ServiceEntry) {
	for entry := range entriesCh {
		log.Debugf("Received entry via fan-in: %v", entry)
		if entry == nil {
			continue
		}

		localEntry := *entry
		m.cache.addEntry(&localEntry)

	}
	log.Debug("fanInCh closed, entry processing finished.")
}
