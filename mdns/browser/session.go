package browser

import (
	"context"
	"sync"

	"github.com/grandcat/zeroconf"
)

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
