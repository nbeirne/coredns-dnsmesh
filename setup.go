package dnsmesh

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

// init registers this plugin.
func init() { plugin.Register("dnsmesh", setup) }

// setup is the function that gets called when the config parser see the token "example". Setup is responsible
// for parsing any extra options the example plugin may have. The first token this function sees is "example".
func setup(c *caddy.Controller) error {
	dns_mesh := &DnsMesh{}
	mesh_providers := []MeshProvider{}

	ts := &TailscaleMeshProvider{}
	ts.tag = "tag:dns"
	for c.Next() {

		args := c.RemainingArgs()
		if len(args) != 1 {
			return plugin.Error("dnsmesh", c.ArgErr())
		}
		dns_mesh.zone = args[0]

		for c.NextBlock() {
			switch c.Val() {
			case "authkey":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return plugin.Error("dnsmesh", c.ArgErr())
				}
				ts.authkey = args[0]
			case "hostname":
				args := c.RemainingArgs()
				if len(args) != 1 {
					return plugin.Error("dnsmesh", c.ArgErr())
				}
				ts.hostname = args[0]
			case "tag": 
				args := c.RemainingArgs()
				if len(args) != 1 {
					return plugin.Error("dnsmesh", c.ArgErr())
				}
				ts.tag = args[0]

			default:
				return plugin.Error("tailscale", c.ArgErr())
			}
		}
	}

	if (ts.tag == "") {
		return plugin.Error("dnsmesh", c.ArgErr())
	}

	mesh_providers = append(mesh_providers, ts)
	dns_mesh.mesh_providers = mesh_providers

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		dns_mesh.next = next
		if err := dns_mesh.start(); err != nil {
			log.Error(err)
			return nil
		}
		return dns_mesh
	})

	// All OK, return a nil error.
	return nil
}
