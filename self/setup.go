package self

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/coredns/caddy"
)

func init() {
	plugin.Register("self", setup)
}

func setup(c *caddy.Controller) error {
	c.Next()
	//if c.NextArg() {
	//	return plugin.Error("self", c.ArgErr())
	//}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return NewSelf(next, c.RemainingArgs())
	})

	return nil
}
