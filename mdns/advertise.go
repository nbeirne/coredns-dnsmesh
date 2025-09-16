package mdns

import (
	"net"

	"github.com/celebdor/zeroconf"
)

type MdnsAdvertise struct {
	advertise    bool
	instanceName string
	service  	 string
	domain   	 string
	port      	 int
	ttl 		 uint32
	txtEntries []string
	ifaceBindSubnet *net.IPNet // subnet to search on

	server 		*zeroconf.Server
}

func NewMdnsAdvertise(instanceName, service string, port int, ttl uint32) *MdnsAdvertise {
	return &MdnsAdvertise{
		advertise:     true,
		instanceName:  instanceName,
		service:  	   service,
		domain:        DefaultDomain, // always use local. Technically this may be different, but resolvers dont generally respect other values.
		port:          port,
		ttl: 		   120,
	}
}

func (m *MdnsAdvertise) BindToSubnet(subnet *net.IPNet) {
	m.ifaceBindSubnet = subnet
}

func (m *MdnsAdvertise) AddTxt(txtEntry string) {
	m.txtEntries = append(m.txtEntries, txtEntry)

	if (m.server != nil) {
		m.server.SetText(m.txtEntries)
	}
}

func (m *MdnsAdvertise) StartAdvertise() error {
	if (m.server != nil) {
		m.StopAdvertise()
	}

	log.Infof("Start advertising... Instance: %s, Service: %s, Port: %d", m.instanceName, m.service, m.port)

	var ifaces []net.Interface
	if m.ifaceBindSubnet != nil {
		foundIfaces, err := FindInterfacesForSubnet(*m.ifaceBindSubnet)
		if err != nil || len(foundIfaces) == 0 {
			log.Errorf("Failed to find interface for '%s'\n", m.ifaceBindSubnet.String())
		} else {
			ifaces = foundIfaces
		}
	}

	server, err := zeroconf.Register(
		m.instanceName, 
		m.service, 
		m.domain, 
		m.port, 
		m.txtEntries,
		ifaces,
	)
	server.TTL(m.ttl) // refresh every 2 mins

	if err != nil {
		log.Errorf("Error staring advertisement: %s", err)
		return err
	}
	m.server = server
	return nil
} 

func (m *MdnsAdvertise) StopAdvertise() {
	log.Infof("Stop advertising...")
	m.server.Shutdown()
}

