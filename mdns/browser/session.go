package browser

import (
	"context"
	"net"
	"sync"

	"github.com/grandcat/zeroconf"
)

// The primary function of ZeroconfSession is to create a session
// which does not close the input entriesCh.
type ZeroconfSession struct {
	zeroConfImpl ZeroconfInterface
	interfaces   *[]net.Interface
}

func NewZeroconfSession(zeroConfImpl ZeroconfInterface, interfaces *[]net.Interface) *ZeroconfSession {
	return &ZeroconfSession{
		zeroConfImpl: zeroConfImpl,
		interfaces:   interfaces,
	}
}

func (zs *ZeroconfSession) Browse(ctx context.Context, service string, domain string, entriesCh chan<- *zeroconf.ServiceEntry) error {
	log.Debug("start browse session....")
	defer log.Debug("end browse session....")

	var wg sync.WaitGroup
	localEntriesCh := make(chan *zeroconf.ServiceEntry)

	// We want to wait for the go routine to finish before stopping the call to Run.
	// Fanning in is a requirement becasue the localEntriesCh is ephemeral and managed by the session.
	wg.Add(1)
	go func() {
		log.Debug("start browse receiver....")
		defer log.Debug("end browse receiver....")
		defer wg.Done()
		for entry := range localEntriesCh {
			log.Debugf("Received entry via Browse: %v", entry)
			entriesCh <- entry
		}
	}()

	// hand off control of localEntriesCh to browseMdns. Its expected to close the channel internally.
	resolver, err := zs.zeroConfImpl.NewResolver(zs.getClientOption())
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", service, err.Error())
		close(localEntriesCh)
		return err
	}

	// ASSUMPTION: resolver.Browse will close localEntriesCh when ctx is cancelled or times out.
	err = resolver.Browse(ctx, service, domain, localEntriesCh)
	if err != nil && ctx.Err() == nil { // Don't log error if it's just a context cancellation
		log.Errorf("Failed to browse %s records: %s", service, err.Error())
		return err
	}

	wg.Wait()
	return err
}

func (zs *ZeroconfSession) Lookup(ctx context.Context, instance string, service string, domain string, entriesCh chan<- *zeroconf.ServiceEntry) error {
	log.Debug("start lookuo session....")
	defer log.Debug("end lookup session....")

	var wg sync.WaitGroup
	localEntriesCh := make(chan *zeroconf.ServiceEntry)

	// We want to wait for the go routine to finish before stopping the call to Run.
	// Fanning in is a requirement becasue the localEntriesCh is ephemeral and managed by the session.
	wg.Add(1)
	go func() {
		log.Debug("start lookup receiver....")
		defer log.Debug("end lookup receiver...")
		defer wg.Done()
		for entry := range localEntriesCh {
			log.Debugf("Received entry via Lookup: %v", entry)
			entriesCh <- entry
		}
	}()

	// hand off control of localEntriesCh to browseMdns. Its expected to close the channel internally.
	resolver, err := zs.zeroConfImpl.NewResolver(zs.getClientOption())
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", service, err.Error())
		close(localEntriesCh)
		return err
	}

	// ASSUMPTION: resolver.Browse will close localEntriesCh when ctx is cancelled or times out.
	err = resolver.Lookup(ctx, instance, service, domain, localEntriesCh)
	if err != nil && ctx.Err() == nil { // Don't log error if it's just a context cancellation
		log.Errorf("Failed to browse %s records: %s", service, err.Error())
		return err
	}

	wg.Wait()
	return err
}

func (zs *ZeroconfSession) getClientOption() zeroconf.ClientOption {
	var opts zeroconf.ClientOption
	if zs.interfaces != nil {
		opts = zeroconf.SelectIfaces(*zs.interfaces)
	}
	return opts
}
