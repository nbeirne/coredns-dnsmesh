
package util

import (
	"net"
	"fmt"
)


func FindInterfacesForSubnet(subnet net.IPNet) (foundIfaces []net.Interface, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return foundIfaces, err
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			currIP, _, err := net.ParseCIDR(addr.String())

			if err != nil {
				continue
			}
			if currIP == nil {
				continue
			}

			if subnet.Contains(currIP) {
				foundIfaces = append(foundIfaces, i)
			}
		}
	}

	return foundIfaces, nil
}


func FindInterfaceForAddress(ip net.IP) (iface net.Interface, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return iface, err
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			currIP, _, err := net.ParseCIDR(addr.String())

			if err != nil {
				continue
			}
			if currIP == nil {
				continue
			}
			if currIP.Equal(ip) {
				iface = i
				return iface, nil
			}
		}
	}

	return iface, fmt.Errorf("Couldn't find interface with IP address %s", ip)
}

