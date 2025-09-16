package mdns

import (
	"errors"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

// TODO: more robust parsing of errors
// TODO: parse fanout options

// init registers this plugin.
func init() { 
	plugin.Register(QueryPluginName, setupQuery) 
	plugin.Register(AdvertisePluginName, setupAdvertise) 
}

// setup is the function that gets called when the config parser see the token "example". Setup is responsible
// for parsing any extra options the example plugin may have. The first token this function sees is "example".
func setupQuery(c *caddy.Controller) error {
	m , err := parseQueryOptions(c)
	if err != nil {
		return err
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		m.Next = next
		if err := m.Start(); err != nil {
			log.Error(err)
			return nil
		}
		return m
	})

	c.OnShutdown(func () error {
		m.browser.Stop()
		return nil
	})
	
	// All OK, return a nil error.
	return nil
}

func setupAdvertise(c *caddy.Controller) error {
	// Defaults
	mdnsType := DefaultServiceType
	ttl := DefaultTTL

	shortHostname, err := getShortHostname()
	if (err != nil) {
		return err
	}
	instanceName := AdvertisingPrefix + shortHostname

	port, err := getServerPort(c)
	if (err != nil) {
		port = 0
	}

	ifaceBindSubnet := (*net.IPNet)(nil)

	c.Next()
	for c.NextBlock() {
		switch c.Val() {
		case "instance_name": 
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.ArgErr()
			}
			instanceName = remaining[0]

		case "type":
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.ArgErr()
			}
			mdnsType = remaining[0]

		case "port":
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.ArgErr()
			}
			portInt, err := strconv.Atoi(remaining[0])
			if err != nil {
				return c.Errf("port provided is invalid: %s", remaining[0])
			}
			port = portInt

		case "ttl":
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.ArgErr()
			}
			ttlInt, err := strconv.ParseUint(remaining[0], 10, 32)
			if err != nil {
				return c.Errf("ttl provided is invalid: %s", remaining[0])
			}
			ttl = uint32(ttlInt)

		case "iface_bind_subnet": 
			remaining := c.RemainingArgs()
			if len(remaining) != 1 {
				return c.ArgErr()
			}
			_, subnet, err := net.ParseCIDR(remaining[0])
			if err != nil {
				return c.Errf("Failed to parse subnet: %s", remaining[0])
			}
			ifaceBindSubnet = subnet

		default:
			return c.Errf("Unknown option: %s", c.Val())
		}
	}

	// TODO: configure
	advertiser := NewMdnsAdvertise(instanceName, mdnsType, port, ttl)
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
		log.Errorf("Error parsing port from address %s", urlStr)
		return 0, err
	}

	port, err := strconv.Atoi(url.Port())

	return port, err
}


func parseQueryOptions(c *caddy.Controller) (*MdnsMeshPlugin, error) {
	m := MdnsMeshPlugin{}

	mdnsType := DefaultServiceType
	ifaceBindSubnet := (*net.IPNet)(nil)

	m.Timeout = DefaultTimeout
	m.addrsPerHost = DefaultAddrsPerHost
	m.addrMode = DefaultAddrMode

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) != 1 {
			return nil, plugin.Error(QueryPluginName, c.ArgErr())
		}
		m.Zone = args[0]

		for c.NextBlock() {
			switch c.Val() {
			case "type": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				mdnsType = remaining[0]

			case "iface_bind_subnet": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				_, subnet, err := net.ParseCIDR(remaining[0])
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("failed to parse subnet: %s", remaining[0]))
				}
				ifaceBindSubnet = subnet

			case "ignore_self": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				ignoreSelf, err := strconv.ParseBool(remaining[0])
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("failed to parse boolean: %s", remaining[0]))
				}
				m.ignoreSelf = ignoreSelf

			case "filter": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				filter, err := regexp.Compile(remaining[0])
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("failed to compile regex: %s", remaining[0]))
				}
				m.filter = filter

			case "address_mode": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}

				switch remaining[0] {
				case "prefer_ipv6": m.addrMode = PreferIPv6
				case "prefer_ipv4": m.addrMode = PreferIPv4
				case "only_ipv6": m.addrMode = IPv6Only
				case "only_ipv4": m.addrMode = IPv4Only
				default:
					return nil, plugin.Error(QueryPluginName, c.Errf("unknown address_mode: %s", remaining[0]))
				}

			case "addresses_per_host": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				addrsPerHostInt, err := strconv.Atoi(remaining[0])
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("addresses_per_host could not be parsed: %s", remaining[0]))
				}
				m.addrsPerHost = addrsPerHostInt

			case "timeout": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				timeout, err := time.ParseDuration(remaining[0])
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("invalid duration for timeout: %s", remaining[0]))
				}
				m.Timeout = timeout

			case "zone": 
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				m.Zone = remaining[0]

			case "attempts":
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				attempts, err := strconv.Atoi(remaining[0])
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("attempts is not an integer: %s", remaining[0]))
				}
				m.Attempts = attempts

			case "worker_count":
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return nil, plugin.Error(QueryPluginName, c.ArgErr())
				}
				workerCount, err := strconv.Atoi(remaining[0])
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("worker_count is not an integer: %s", remaining[0]))
				}
				m.WorkerCount = workerCount

			default:
				return nil, plugin.Error(QueryPluginName, c.Errf("unknown option: %s", c.Val()))
			}
		}
	}

	browser := NewMdnsBrowser(mdnsType, ifaceBindSubnet)
	m.browser = &browser

	return &m, nil
}

// TODO: test parsing:
//:53
//127.0.0.1:1053
//[::1]:1053
//example.org (no explicit port; defaults to 53)
