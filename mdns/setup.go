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

	"github.com/nbeirne/coredns-dnsmesh/mdns/browser"
)

// TODO: more robust parsing of errors
// TODO: parse fanout options

// init registers this plugin.
func init() {
	plugin.Register(QueryPluginName, setupQuery)
	plugin.Register(AdvertisePluginName, setupAdvertise)
}

type interfaceFinder func(net.IPNet) ([]net.Interface, error)

// setup is the function that gets called when the config parser see the token "example". Setup is responsible
// for parsing any extra options the example plugin may have. The first token this function sees is "example".
func setupQuery(c *caddy.Controller) error {
	m, err := parseQueryOptions(c, FindInterfacesForSubnet)
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

	c.OnShutdown(func() error {
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
	if err != nil {
		return err
	}
	instanceName := AdvertisingPrefix + shortHostname

	port, err := getServerPort(c)
	if err != nil {
		port = 0
	}

	ifaceBindSubnet := (*net.IPNet)(nil)

	c.Next()
	for c.NextBlock() {
		switch c.Val() {
		case "instance_name":
			val, err := parseSingleArg(c)
			if err != nil {
				return err
			}
			instanceName = val

		case "type":
			val, err := parseSingleArg(c)
			if err != nil {
				return err
			}
			mdnsType = val

		case "port":
			val, err := parseSingleArg(c)
			if err != nil {
				return err
			}
			portInt, err := strconv.Atoi(val)
			if err != nil {
				return c.Errf("port provided is invalid: %s", val)
			}
			port = portInt

		case "ttl":
			val, err := parseSingleArg(c)
			if err != nil {
				return err
			}
			ttlInt, err := strconv.ParseUint(val, 10, 32)
			if err != nil {
				return c.Errf("ttl provided is invalid: %s", val)
			}
			ttl = uint32(ttlInt)

		case "iface_bind_subnet":
			val, err := parseSingleArg(c)
			if err != nil {
				return err
			}
			_, subnet, err := net.ParseCIDR(val)
			if err != nil {
				return c.Errf("Failed to parse subnet: %s", val)
			}
			ifaceBindSubnet = subnet

		default:
			return c.Errf("Unknown option: %s", c.Val())
		}
	}

	// TODO: configure
	advertiser := NewMdnsAdvertise(instanceName, mdnsType, port, ttl)
	advertiser.BindToSubnet(ifaceBindSubnet)

	c.OnStartup(func() error {
		return advertiser.StartAdvertise()
	})

	c.OnShutdown(func() error {
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
		return 0, errors.New("error fetching port from server block")
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

func parseSingleArg(c *caddy.Controller) (string, error) {
	optionName := c.Val()

	if !c.NextArg() {
		return "", c.Errf("option '%s' expects an argument, but got none was provided", optionName)
	}

	val := c.Val()
	if val == "{" || val == "}" {
		return "", c.Errf("option '%s' expects an argument, but got '%v'", optionName, val)
	}
	//	if c.NextArg() {
	//		return "", c.Errf("option '%s' expects only one argument, but found more: '%s'", optionName, c.Val())
	//	}
	return val, nil
}

func parseQueryOptions(c *caddy.Controller, findIfaces interfaceFinder) (*MdnsMeshPlugin, error) {
	m := MdnsMeshPlugin{}

	mdnsType := DefaultServiceType
	ifaceBindSubnet := (*net.IPNet)(nil)

	m.Timeout = DefaultTimeout
	m.addrsPerHost = DefaultAddrsPerHost
	m.addrMode = DefaultAddrMode

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) < 1 {
			return nil, plugin.Error(QueryPluginName, c.Errf("a zone must be specified"))
		}
		m.Zone = args[0]

		for c.NextBlock() {
			switch c.Val() {
			case "type":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				mdnsType = val

			case "iface_bind_subnet":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				_, subnet, err := net.ParseCIDR(val)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("failed to parse subnet: %s", val))
				}
				ifaceBindSubnet = subnet

			case "ignore_self":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				ignoreSelf, err := strconv.ParseBool(val)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("failed to parse boolean for 'ignore_self': %s", val))
				}
				m.ignoreSelf = ignoreSelf

			case "filter":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				filter, err := regexp.Compile(val)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("failed to compile regex for 'filter': %s", val))
				}
				m.filter = filter

			case "address_mode":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				switch val {
				case "prefer_ipv6":
					m.addrMode = PreferIPv6
				case "prefer_ipv4":
					m.addrMode = PreferIPv4
				case "only_ipv6":
					m.addrMode = IPv6Only
				case "only_ipv4":
					m.addrMode = IPv4Only
				default:
					return nil, plugin.Error(QueryPluginName, c.Errf("unknown address_mode: %s", val))
				}

			case "addresses_per_host":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				addrsPerHostInt, err := strconv.Atoi(val)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("addresses_per_host could not be parsed: %s", val))
				}
				m.addrsPerHost = addrsPerHostInt

			case "timeout":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				timeout, err := time.ParseDuration(val)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("invalid duration for timeout: %s", val))
				}
				m.Timeout = timeout

			case "zone":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				m.Zone = val

			case "attempts":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				attempts, err := strconv.Atoi(val)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("attempts is not an integer: %s", val))
				}
				m.Attempts = attempts

			case "worker_count":
				val, err := parseSingleArg(c)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, err)
				}
				workerCount, err := strconv.Atoi(val)
				if err != nil {
					return nil, plugin.Error(QueryPluginName, c.Errf("worker_count is not an integer: %s", val))
				}
				m.WorkerCount = workerCount

			default:
				return nil, plugin.Error(QueryPluginName, c.Errf("unknown option: %s", c.Val()))
			}
		}
	}

	var ifaces *[]net.Interface
	if ifaceBindSubnet != nil {
		foundIfaces, err := findIfaces(*ifaceBindSubnet)
		if err != nil || len(foundIfaces) == 0 {
			log.Errorf("Failed to find interface for '%s'\n", ifaceBindSubnet.String())
			ifaces = &([]net.Interface{})
		} else {
			ifaces = &foundIfaces
		}
	}

	browser := browser.NewZeroconfBrowser("local.", mdnsType, ifaces)
	browser.Log = log
	m.browser = browser

	return &m, nil
}

// TODO: test parsing:
//:53
//127.0.0.1:1053
//[::1]:1053
//example.org (no explicit port; defaults to 53)
