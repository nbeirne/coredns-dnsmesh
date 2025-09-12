package dnsmesh

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

// init registers this plugin.
func init() { plugin.Register("dnsmesh_tailscale", setup) }

// setup is the function that gets called when the config parser see the token "example". Setup is responsible
// for parsing any extra options the example plugin may have. The first token this function sees is "example".
func setup(c *caddy.Controller) error {
	d := &DnsMesh{}
	d.meshProviders = []MeshProvider{}

	for c.Next() {

		args := c.RemainingArgs()
		if len(args) != 1 {
			return plugin.Error("dnsmesh_tailscale", c.ArgErr())
		}
		d.Zone = args[0]

		for c.NextBlock() {
			switch c.Val() {

		//	case "fallthrough": // TODO: fallthrough
		//		d.fall.SetZonesFromArgs(c.RemainingArgs())

			default:
				return plugin.Error("tailscale", c.ArgErr())
			}
		}
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		d.Next = next
		if err := d.Start(); err != nil {
			log.Error(err)
			return nil
		}
		return d
	})

	// All OK, return a nil error.
	return nil
}
