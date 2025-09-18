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

	mutex           *sync.RWMutex
	cancelBrowse 	context.CancelFunc

	services        *map[string]*trackedService
}

func NewZeroconfBrowser(domain, mdnsType string, interfaces *[]net.Interface) (browser ZeroconfBrowser) {
	browser = ZeroconfBrowser{}
	browser.mdnsType = mdnsType
	browser.domain = domain
	browser.interfaces = interfaces

	browser.zeroConfImpl = ZeroconfImpl{}

	services := make(map[string]*trackedService)
	browser.services = &services

	mutex := sync.RWMutex{}
	browser.mutex = &mutex

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
		m.mutex.RLock()
		cancel := m.cancelBrowse
		m.mutex.RUnlock()
		if cancel != nil {
			cancel()
		}

		m.wg.Wait()
		log.Info("Stopped MdnsBrowser.")
	})
}

func (m *ZeroconfBrowser) Services() []*zeroconf.ServiceEntry {
	now := time.Now()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	serviceEntries := make([]*zeroconf.ServiceEntry, 0, len(*m.services))

	for _, s := range *m.services {
		if now.After(s.expiry) {
			continue
		}
		serviceEntries = append(serviceEntries, s.entry)
	}
	return serviceEntries
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
			m.addEntry(&localEntry)

			log.Debug("Recalculate signal received, resetting timer.")
			if !timer.Stop() {
				<-timer.C // Drain the timer if it already fired.
			}
			refreshInterval := m.calculateNextRefresh()
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
			m.runBrowseSession(sessionCtx, &sessionWg, fanInCh)

			// After the session, remove expired services and reset the timer for the next session.
			m.removeExpiredServices()
			refreshInterval := m.calculateNextRefresh()
			log.Debugf("Next mDNS refresh scheduled in %v", refreshInterval)
			timer.Reset(refreshInterval)
		}
	}
}

func (m *ZeroconfBrowser) runBrowseSession(ctx context.Context, sessionWg *sync.WaitGroup, fanInCh chan<- *zeroconf.ServiceEntry) {
	sessionWg.Add(1)
	// This manager goroutine waits for the browse and forwarder to complete,
	// then signals the session is done via the WaitGroup.
	go func() {
		log.Debug("start browse session....")
		defer log.Debug("end browse session....")
		defer sessionWg.Done()
		var wg sync.WaitGroup
		wg.Add(2) // One for browseMdns, one for the forwarder.
		localEntriesCh := make(chan *zeroconf.ServiceEntry)

		go func() {
			log.Debug("start browse mdns....")
			defer log.Debug("end browse mdns....")	
			defer wg.Done()
			m.browseMdns(ctx, localEntriesCh)
		}()

		go func() {
			log.Debug("start fan in ch....")
			defer log.Debug("end fan in ch....")	
			defer wg.Done()
			for entry := range localEntriesCh {
				log.Debugf("Received entry via chan: %v", entry)
				fanInCh <- entry
			}
		}()

		wg.Wait()
	}()
}

func (m *ZeroconfBrowser) browseMdns(ctx context.Context, entriesCh chan *zeroconf.ServiceEntry) error {
	log.Debugf("browseMdns... Starting.")
	defer log.Debugf("browseMdns... Finished.")

	var opts zeroconf.ClientOption
	if m.interfaces != nil {
		opts = zeroconf.SelectIfaces(*m.interfaces)
	}
	resolver, err := m.zeroConfImpl.NewResolver(opts)
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", m.mdnsType, err.Error())
		return err
	}

	err = resolver.Browse(ctx, m.mdnsType, m.domain, entriesCh)
	if err != nil {
		log.Errorf("Failed to browse %s records: %s", m.mdnsType, err.Error())
		return err
	}

	return nil
}
