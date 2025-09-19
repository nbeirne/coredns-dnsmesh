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

func (zs *ZeroconfSession) Run(ctx context.Context, entriesCh chan<- *zeroconf.ServiceEntry) error {
	log.Debug("start browse session....")
	defer log.Debug("end browse session....")

	var wg sync.WaitGroup
	localEntriesCh := make(chan *zeroconf.ServiceEntry)

	// We want to wait for the go routine to finish before stopping the call to Run.
	// Fanning in is a requirement becasue the localEntriesCh is ephemeral and managed by the session.
	wg.Add(1)
	go func() {
		log.Debug("start fan in ch....")
		defer log.Debug("end fan in ch....")
		defer wg.Done()
		for entry := range localEntriesCh {
			log.Debugf("Received entry via chan: %v", entry)
			entriesCh <- entry
		}
	}()

	// hand off control of localEntriesCh to browseMdns. Its expected to close the channel internally.
	err := zs.browseMdns(ctx, localEntriesCh)
	wg.Wait()
	return err
}

func (zs *ZeroconfSession) browseMdns(ctx context.Context, localEntriesCh chan<- *zeroconf.ServiceEntry) error {
	log.Debugf("browseMdns... Starting.")
	defer log.Debugf("browseMdns... Finished.")

	resolver, err := zs.zeroConfImpl.NewResolver(zs.getClientOption())
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", zs.mdnsType, err.Error())
		close(localEntriesCh)
		return err
	}

	// ASSUMPTION: resolver.Browse will close localEntriesCh when ctx is cancelled or times out.
	err = resolver.Browse(ctx, zs.mdnsType, zs.domain, localEntriesCh)
	if err != nil && ctx.Err() == nil { // Don't log error if it's just a context cancellation
		log.Errorf("Failed to browse %s records: %s", zs.mdnsType, err.Error())
		return err
	}

	return nil
}

func (zs *ZeroconfSession) getClientOption() zeroconf.ClientOption {
	var opts zeroconf.ClientOption
	if zs.interfaces != nil {
		opts = zeroconf.SelectIfaces(*zs.interfaces)
	}
	return opts
}
