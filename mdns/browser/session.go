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
	resolver, err := zs.zeroConfImpl.NewResolver(zs.getClientOption())
	if err != nil {
		// No goroutine was started, so we can return directly.
		return err
	}

	var wg sync.WaitGroup
	localEntriesCh := make(chan *zeroconf.ServiceEntry)

	// We want to wait for the go routine to finish before stopping the call to Run.
	// Fanning in is a requirement becasue the localEntriesCh is ephemeral and managed by the session.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range localEntriesCh {
			localEntry := *entry // make copy of entry so that zeroconf does not edit it from under us
			entriesCh <- &localEntry
		}
	}()

	// ASSUMPTION: resolver.Browse will close localEntriesCh when ctx is cancelled or times out.
	err = resolver.Browse(ctx, service, domain, localEntriesCh)
	if err != nil && ctx.Err() == nil { // Don't log error if it's just a context cancellation
		return err
	}

	wg.Wait()
	return err
}

func (zs *ZeroconfSession) Lookup(ctx context.Context, instance string, service string, domain string, entriesCh chan<- *zeroconf.ServiceEntry) error {
	resolver, err := zs.zeroConfImpl.NewResolver(zs.getClientOption())
	if err != nil {
		// No goroutine was started, so we can return directly.
		return err
	}

	var wg sync.WaitGroup
	localEntriesCh := make(chan *zeroconf.ServiceEntry)

	// We want to wait for the go routine to finish before stopping the call to Run.
	// Fanning in is a requirement becasue the localEntriesCh is ephemeral and managed by the session.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for entry := range localEntriesCh {
			localEntry := *entry // make copy of entry so that zeroconf does not edit it from under us
			entriesCh <- &localEntry
		}
	}()

	// ASSUMPTION: resolver.Browse will close localEntriesCh when ctx is cancelled or times out.
	err = resolver.Lookup(ctx, instance, service, domain, localEntriesCh)
	if err != nil && ctx.Err() == nil { // Don't log error if it's just a context cancellation
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
