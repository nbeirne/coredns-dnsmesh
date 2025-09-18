package browser

import (
	"context"
	"net"
	"sync"

	"github.com/grandcat/zeroconf"
)

type ZeroconfSession struct {
	zeroConfImpl ZeroconfInterface
	interfaces   *[]net.Interface
	mdnsType     string
	domain       string
}

// NewZeroconfSession creates a new ZeroconfSession object.
func NewZeroconfSession(zeroConfImpl ZeroconfInterface, interfaces *[]net.Interface, mdnsType, domain string) *ZeroconfSession {
	return &ZeroconfSession{
		zeroConfImpl: zeroConfImpl,
		interfaces:   interfaces,
		mdnsType:     mdnsType,
		domain:       domain,
	}
}

func (zs *ZeroconfSession) Run(ctx context.Context, sessionWg *sync.WaitGroup, fanInCh chan<- *zeroconf.ServiceEntry) {
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
			zs.browseMdns(ctx, localEntriesCh)
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

func (zs *ZeroconfSession) browseMdns(ctx context.Context, entriesCh chan *zeroconf.ServiceEntry) error {
	log.Debugf("browseMdns... Starting.")
	defer log.Debugf("browseMdns... Finished.")

	var opts zeroconf.ClientOption
	if zs.interfaces != nil {
		opts = zeroconf.SelectIfaces(*zs.interfaces)
	}
	resolver, err := zs.zeroConfImpl.NewResolver(opts)
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", zs.mdnsType, err.Error())
		return err
	}

	err = resolver.Browse(ctx, zs.mdnsType, zs.domain, entriesCh)
	if err != nil && ctx.Err() == nil { // Don't log error if it's just a context cancellation
		log.Errorf("Failed to browse %s records: %s", zs.mdnsType, err.Error())
		return err
	}

	return nil
}
