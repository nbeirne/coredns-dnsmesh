package mdns

import (
	"github.com/nbeirne/coredns-dnsmesh/util"

	"context"
	"time"
	"strings"
	"net"
	"sync"

	//"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	clog "github.com/coredns/coredns/plugin/pkg/log"

	"github.com/celebdor/zeroconf"
	//"github.com/openshift/mdns-publisher/pkg/publisher" // TODO: publish??
)

type MdnsProvider struct {
	dnsMesh util.DnsMesh

	mdnsType    string
	filter      string // TODO: filter???

	mutex       *sync.RWMutex
	mdnsHosts   *map[string]*zeroconf.ServiceEntry
	hosts []util.DnsHost
}

var log = clog.NewWithPlugin("dnsmesh_mdns")

// Name implements the Handler interface.
func (m *MdnsProvider) Name() string { return "dnsmesh_mdns" }

func (m *MdnsProvider) Start() error {
	m.dnsMesh.AddMeshProvider(m)

	m.mdnsType = "_airplay._tcp" // TODO: configure

	mdnsHosts := make(map[string]*zeroconf.ServiceEntry)
	mutex := sync.RWMutex{}

	m.mdnsHosts = &mdnsHosts
	m.mutex = &mutex // TODO: ptr??

	go browseLoop(m)
	// start browsing for mdns servers

	return nil
}

func (m *MdnsProvider) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Infof("Received request for name: %v", r.Question[0].Name)

	f := m.dnsMesh.CreateFanout()

	return f.ServeDNS(ctx, w, r)
}

func (m *MdnsProvider) MeshDnsHosts() []util.DnsHost {
	log.Infof("create mesh host.. providers: %s", m.dnsMesh.MeshProviders)
	return m.hosts
}



func browseLoop(m *MdnsProvider) {
	for {
		m.BrowseMDNS()
		// 5 seconds seems to be the minimum ttl that the cache plugin will allow
		// Since each browse operation takes around 2 seconds, this should be fine
		// TODO: caching strategy????
		time.Sleep(5 * time.Second)
	}
}

func (m *MdnsProvider) BrowseMDNS() {
	entriesCh := make(chan *zeroconf.ServiceEntry)
	mdnsHosts := make(map[string]*zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		log.Debug("Retrieving mDNS entries")
		for entry := range results {
			// Make a copy of the entry so zeroconf can't later overwrite our changes
			localEntry := *entry
			log.Debugf("A Instance: %s, HostName: %s, AddrIPv4: %s, AddrIPv6: %s Port: %d\n", localEntry.Instance, localEntry.HostName, localEntry.AddrIPv4, localEntry.AddrIPv6, localEntry.Port)
			if strings.Contains(localEntry.Instance, m.filter) {
				mdnsHosts[localEntry.HostName] = entry
			} else {
				log.Debugf("Ignoring entry '%s' because it doesn't match filter '%s'\n",
					localEntry.Instance, m.filter)
			}
		}
	}(entriesCh)

	var iface net.Interface
	//if m.bindAddress != "" { // TODO: bind???
	//	foundIface, err := publisher.FindIface(net.ParseIP(m.bindAddress))
	//	if err != nil {
	//		log.Errorf("Failed to find interface for '%s'\n", m.bindAddress)
	//	} else {
	//		iface = foundIface
	//	}
	//}
	_ = queryService(m.mdnsType, entriesCh, iface, ZeroconfImpl{})

	m.mutex.Lock()
	defer m.mutex.Unlock()
	// Clear maps so we don't have stale entries
	for k := range *m.mdnsHosts {
		delete(*m.mdnsHosts, k)
	}
	// Copy values into the shared maps only after we've collected all of them.
	// This prevents us from having to lock during the entire mdns browse time.
	for k, v := range mdnsHosts {
		(*m.mdnsHosts)[k] = v
	}
	log.Infof("mdnsHosts: %v", m.mdnsHosts)
	for name, entry := range *m.mdnsHosts {
		log.Debugf("%s: %v", name, entry)
	}
}

func queryService(service string, channel chan *zeroconf.ServiceEntry, iface net.Interface, z ZeroconfInterface) error {
	var opts zeroconf.ClientOption
	if iface.Name != "" {
		opts = zeroconf.SelectIfaces([]net.Interface{iface})
	}
	resolver, err := z.NewResolver(opts)
	if err != nil {
		log.Errorf("Failed to initialize %s resolver: %s", service, err.Error())
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
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

