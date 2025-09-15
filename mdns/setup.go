package mdns

import (
	"errors"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/nbeirne/coredns-dnsmesh/util"
)

// init registers this plugin.
func init() { 
	plugin.Register(QueryPluginName, setupQuery) 
	plugin.Register(AdvertisePluginName, setupAdvertise) 
}

// setup is the function that gets called when the config parser see the token "example". Setup is responsible
// for parsing any extra options the example plugin may have. The first token this function sees is "example".
func setupQuery(c *caddy.Controller) error {
	t := &MdnsProvider{}

	mdnsType := DefaultServiceType
	ifaceBindSubnet := (*net.IPNet)(nil)

	t.dnsMesh = util.DnsMesh{}
	t.addrsPerHost = DefaultAddrsPerHost
	t.addrMode = DefaultAddrMode

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) != 1 {
			return plugin.Error(QueryPluginName, c.ArgErr())
		}
		t.dnsMesh.Zone = args[0]

		for c.NextBlock() {
			switch c.Val() {
			case "type": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return c.Errf("type needs to exist")
				}
				mdnsType = remaining[0]

			case "iface_bind_subnet": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return c.Errf("iface_bind_subnet needs to exist")
				}
				_, subnet, err := net.ParseCIDR(remaining[0])
				if err != nil {
					return c.Errf("Failed to parse subnet: %w", err)
				}
				ifaceBindSubnet = subnet

			case "filter": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return c.Errf("filter needs to exist")
				}
				filter, err := regexp.Compile(remaining[0])
				if err != nil {
					return c.Errf("Failed to compile regex: %w", err)
				}
				t.filter = filter

			case "address_mode": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return c.Errf("address_mode needs to exist")
				}

				switch remaining[0] {
				case "prefer_ipv6": t.addrMode = PreferIPv6
				case "prefer_ipv4": t.addrMode = PreferIPv4
				case "only_ipv6": t.addrMode = IPv6Only
				case "only_ipv4": t.addrMode = IPv4Only
				}

			case "addresses_per_host": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return c.Errf("addresses_per_host needs to exist")
				}
				addrsPerHostInt, err := strconv.Atoi(remaining[0])
				if err != nil {
					return c.Errf("addresses_per_host could not be parsed: %w", err)
				}
				t.addrsPerHost = addrsPerHostInt

			default:
				return c.Errf("Unknown option: %s", c.Val())
			}
		}
	}

	browser := NewMdnsBrowser(mdnsType, ifaceBindSubnet)
	t.browser = &browser

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		t.dnsMesh.Next = next
		if err := t.Start(); err != nil {
			log.Error(err)
			return nil
		}
		return t
	})

	c.OnShutdown(func () error {
		browser.Stop()
		return nil
	})
	

	// All OK, return a nil error.
	return nil
}

func setupAdvertise(c *caddy.Controller) error {
	// Defaults
	mdnsType := DefaultServiceType

	shortHostname, err := getShortHostname()
	if (err != nil) {
		return err
	}
	instanceName := AdvertisingPrefix + shortHostname

	port, err := getServerPort(c)
	if (err != nil) {
		return err
	}

	ifaceBindSubnet := (*net.IPNet)(nil)

	c.Next()
	for c.NextBlock() {
		switch c.Val() {
		case "instanceName": 
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.Errf("instanceName needs to exist")
			}
			instanceName = remaining[0]

		case "type":
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.Errf("type needs to exist")
			}
			mdnsType = remaining[0]

		case "port":
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.Errf("port needs to exist")
			}
			portInt, err := strconv.Atoi(remaining[0])
			if err != nil {
				return c.Errf("port provided is invalid: %w", err)
			}
			port = portInt

		case "iface_bind_subnet": 
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.Errf("type needs to exist")
			}
			_, subnet, err := net.ParseCIDR(remaining[0])
			if err != nil {
				return c.Errf("Failed to parse subnet: %w", err)
			}
			ifaceBindSubnet = subnet

		default:
			return c.Errf("Unknown option: %s", c.Val())
		}
	}

	// TODO: configure
	advertiser := NewMdnsAdvertise(instanceName, mdnsType, port)
	advertiser.BindToSubnet(ifaceBindSubnet)

	c.OnStartup(func () error {
		return advertiser.StartAdvertise()
	}) 

	c.OnShutdown(func () error {
		advertiser.StopAdvertise()
		return nil
	}) 

	return nil
}

func getShortHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	shortName := strings.Split(hostname, ".")[0]
	return shortName, nil
}

func getServerPort(c *caddy.Controller) (int, error) {
	keys := c.ServerBlockKeys
	if len(keys) == 0 {
		return 0, errors.New("Error fetching port from server block...")
	}
	log.Debugf("got key: %v", keys)

	urlStr := keys[0]
	url, err := url.Parse(urlStr)
	if err != nil {
		log.Errorf("Error parsing port from address %s: %w", urlStr, err)
		return 0, err
	}

	port, err := strconv.Atoi(url.Port())

	return port, err
}

// TODO: test parsing:
//:53
//127.0.0.1:1053
//[::1]:1053
//example.org (no explicit port; defaults to 53)

