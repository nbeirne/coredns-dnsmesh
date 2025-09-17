package browser

import (
	"context"
	"maps"
	"net"
	"sync"
	"slices"
	"time"

	"github.com/celebdor/zeroconf"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("mdns_browser")

type MdnsBrowserInterface interface {
	Start() error
	Stop()
	Services() []*zeroconf.ServiceEntry
}

type MdnsBrowser struct {
	domain 			 string
	mdnsType       	 string
	ifaceBindSubnet *net.IPNet // subnet to search on

	zeroConfImpl     ZeroconfInterface

	startOnce        sync.Once
	stopCh           chan struct{}

	mutex           *sync.RWMutex
	cancelBrowse 	context.CancelFunc
	services        *map[string]*zeroconf.ServiceEntry
}

func NewMdnsBrowser(domain, mdnsType string, ifaceBindSubnet *net.IPNet) (browser MdnsBrowser) {
	browser = MdnsBrowser{}
	browser.mdnsType = mdnsType
	browser.domain = domain
	browser.ifaceBindSubnet = ifaceBindSubnet

	browser.zeroConfImpl = ZeroconfImpl{}

	services := make(map[string]*zeroconf.ServiceEntry)
	browser.services = &services

	mutex := sync.RWMutex{}
	browser.mutex = &mutex
	browser.stopCh = make(chan struct{})

	return browser
}

func (m *MdnsBrowser) Start() {
	log.Info("Starting mDNS browser...")
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

	services := slices.Collect(maps.Values(*(m.services)))
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

func (m *MdnsBrowser) receiveEntries(entriesCh <- chan *zeroconf.ServiceEntry) {
	log.Debug("Retrieving mDNS entries")
	for entry := range entriesCh {
		// Make a copy of the entry so zeroconf can't later overwrite our changes
		localEntry := *entry
		log.Infof("Service Instance: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d\n", localEntry.Instance, localEntry.HostName, localEntry.AddrIPv4, localEntry.AddrIPv6, localEntry.Port, localEntry.TTL)

		m.mutex.Lock()
		(*m.services)[localEntry.Instance] = &localEntry
		m.mutex.Unlock()
	}
}

func (m *MdnsBrowser) browseMdns() {
	log.Debugf("Browse starting...")
	entriesCh := make(chan *zeroconf.ServiceEntry)
	go m.receiveEntries(entriesCh)

	var ifaces *[]net.Interface
	if m.ifaceBindSubnet != nil {
		foundIfaces, err := FindInterfacesForSubnet(*m.ifaceBindSubnet)
		if err != nil || len(foundIfaces) == 0 {
			log.Errorf("Failed to find interface for '%s'\n", m.ifaceBindSubnet.String())
			ifaces = &([]net.Interface{})
		} else {
			ifaces = &foundIfaces
		}
	}

	_ = m.queryService(m.mdnsType, entriesCh, ifaces, m.zeroConfImpl)

	log.Infof("Browse finished...")
}



func (m *MdnsBrowser) queryService(service string, channel chan *zeroconf.ServiceEntry, ifaces *[]net.Interface, z ZeroconfInterface) error {
	var opts zeroconf.ClientOption
	if ifaces != nil {
		opts = zeroconf.SelectIfaces(*ifaces)
	}
	resolver, err := z.NewResolver(opts)
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", service, err.Error())
		return err
	}

	//ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.cancelBrowse = cancel

	err = resolver.Browse(ctx, service, m.domain, channel)
	if err != nil {
		log.Errorf("Failed to browse %s records: %s", service, err.Error())
		return err
	}

	// wait until cancel or timeout
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
