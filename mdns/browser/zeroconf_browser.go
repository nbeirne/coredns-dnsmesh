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
	"time"

	"github.com/grandcat/zeroconf"
)

type ZeroconfBrowser struct {
	domain 			 string
	mdnsType       	 string
	interfaces 		*[]net.Interface // subnet to search on

	zeroConfImpl     ZeroconfInterface
	
	startOnce        sync.Once
	stopOnce         sync.Once
	wg               sync.WaitGroup

	cancelBrowse 	context.CancelFunc
	cache            *serviceCache
}

func NewZeroconfBrowser(domain, mdnsType string, interfaces *[]net.Interface) (browser ZeroconfBrowser) {
	browser = ZeroconfBrowser{}
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

	timer := time.NewTimer(0) // Fire immediately for the first browse
	var sessionWg sync.WaitGroup
	var cancelCurrentSession context.CancelFunc
	
	fanInCh := make(chan *zeroconf.ServiceEntry, 10)

	// This goroutine handles processing entries and shutting down.
	go func() {
		defer m.wg.Done()
		for entry := range fanInCh {
			log.Debugf("Received entry via fan-in: %v", entry)
			if entry == nil {
				continue
			}

			localEntry := *entry
			m.cache.addEntry(&localEntry)

			log.Debug("Recalculate signal received, resetting timer.")
			if !timer.Stop() {
				// <-timer.C // Drain the timer if it already fired.
			}
			refreshInterval := m.cache.calculateNextRefresh()
			log.Debugf("Next mDNS refresh scheduled in %v", refreshInterval)
			timer.Reset(refreshInterval)
		}
		log.Debug("fanInCh closed, entry processing finished.")
	}()

	for {
		select {
			
		case <-outerCtx.Done():
			if cancelCurrentSession != nil {
				cancelCurrentSession()
			}
			sessionWg.Wait() // Wait for the final session to exit cleanly.
			close(fanInCh)   // Close fanInCh only after all writers are done.
			log.Debug("browseLoop: Stop signal received, cleaned up.")
			return

		case <-timer.C:
			log.Debug("Refresh timer fired. Starting new mDNS browse.")
			if cancelCurrentSession != nil {
				cancelCurrentSession() // Signal the previous session to stop.
				sessionWg.Wait()       // Wait for it to finish completely.
			}

			// Start a new non-blocking browse session.
			var sessionCtx context.Context
			sessionCtx, cancelCurrentSession = context.WithCancel(outerCtx)
			NewZeroconfSession(m.zeroConfImpl, m.interfaces, m.mdnsType, m.domain).Run(sessionCtx, &sessionWg, fanInCh)

			// After the session, remove expired services and reset the timer for the next session.
			m.cache.removeExpiredServices()
			refreshInterval := m.cache.calculateNextRefresh()
			log.Debugf("Next mDNS refresh scheduled in %v", refreshInterval)
			timer.Reset(refreshInterval)
		}
	}
}
