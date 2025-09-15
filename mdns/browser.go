package mdns

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/celebdor/zeroconf"

	"github.com/nbeirne/coredns-dnsmesh/util"
)


type MdnsBrowserInterface interface {
	Start() error
	Stop()
	Services() []*zeroconf.ServiceEntry
}

type MdnsBrowser struct {
	mdnsType       	 string
	ifaceBindSubnet *net.IPNet // subnet to search on
	startOnce       sync.Once
	stopCh          chan struct{}

	mutex           *sync.RWMutex
	services      []*zeroconf.ServiceEntry
}

func NewMdnsBrowser(mdnsType string, ifaceBindSubnet *net.IPNet) (browser MdnsBrowser) {
	browser = MdnsBrowser{}
	browser.mdnsType = mdnsType
	browser.ifaceBindSubnet = ifaceBindSubnet

	browser.services = []*zeroconf.ServiceEntry{}
	mutex := sync.RWMutex{}
	browser.mutex = &mutex
	browser.stopCh = make(chan struct{})

	return browser
}

func (m *MdnsBrowser) Start() {
	m.startOnce.Do(func() {
		go m.browseLoop()
	})
}

func (m *MdnsBrowser) Stop() {
	log.Info("Stopping mDNS browser...")
	close(m.stopCh)
}

func (m *MdnsBrowser) Services() []*zeroconf.ServiceEntry {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	services := m.services
	return services
}

func (m *MdnsBrowser) browseLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.browseMdns()
		}
	}
}

func (m *MdnsBrowser) browseMdns() {
	entriesCh := make(chan *zeroconf.ServiceEntry)
	mdnsServices := []*zeroconf.ServiceEntry {}
	go func(results <-chan *zeroconf.ServiceEntry) {
		log.Debug("Retrieving mDNS entries")
		for entry := range results {
			// Make a copy of the entry so zeroconf can't later overwrite our changes
			localEntry := *entry
			log.Debugf("Service Instance: %s, HostName: %s, AddrIPv4: %s, AddrIPv6: %s Port: %d, TTL: %d\n", localEntry.Instance, localEntry.HostName, localEntry.AddrIPv4, localEntry.AddrIPv6, localEntry.Port, localEntry.TTL)
			mdnsServices = append(mdnsServices, &localEntry)
		}
	}(entriesCh)

	var ifaces []net.Interface
	if m.ifaceBindSubnet != nil {
		foundIfaces, err := util.FindInterfacesForSubnet(*m.ifaceBindSubnet)
		if err != nil || len(foundIfaces) == 0 {
			log.Errorf("Failed to find interface for '%s'\n", m.ifaceBindSubnet.String())
		} else {
			ifaces = foundIfaces
		}
	}
	_ = queryService(m.mdnsType, entriesCh, ifaces, ZeroconfImpl{})

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Copy values into the shared maps only after we've collected all of them.
	// This prevents us from having to lock during the entire mdns browse time.
	m.services = mdnsServices
	log.Infof("Found %d mdns hosts", len(m.services))
}



func queryService(service string, channel chan *zeroconf.ServiceEntry, ifaces []net.Interface, z ZeroconfInterface) error {
	var opts zeroconf.ClientOption
	if len(ifaces) != 0 {
		opts = zeroconf.SelectIfaces(ifaces)
	}
	resolver, err := z.NewResolver(opts)
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", service, err.Error())
		return err
	}
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	//err = resolver.Browse(context.Background(), service, "local.", channel)
	err = resolver.Browse(ctx, service, "local.", channel)
	if err != nil {
		log.Errorf("Failed to browse %s records: %s", service, err.Error())
		return err
	}
	<-ctx.Done()
	return nil
}



// allow for mocking in tests
type ZeroconfInterface interface {
	NewResolver(...zeroconf.ClientOption) (ResolverInterface, error)
}

type ZeroconfImpl struct{}

func (z ZeroconfImpl) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	return zeroconf.NewResolver(opts...)
}

type ResolverInterface interface {
	Browse(context.Context, string, string, chan<- *zeroconf.ServiceEntry) error
}
