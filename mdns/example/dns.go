package main

import (
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/view"
	_ "github.com/coredns/coredns/plugin/debug"
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/whoami"
	_ "github.com/coredns/coredns/plugin/reload"

	"github.com/coredns/coredns/core/dnsserver"
	_ "github.com/coredns/records"
	"github.com/coredns/coredns/coremain"

	_ "github.com/networkservicemesh/fanout"

	_ "github.com/nbeirne/coredns-dnsmesh/mdns"
)

var directives = []string{
	"reload",
	"debug",
	"errors",
	"log",
	"cache",
	"hosts",
	"dnsmesh_mdns_forward",
	"dnsmesh_mdns_advertise",
	"forward",
	"whoami",
	"view",
}

func init() {
	dnsserver.Directives = directives
}

func main() {
	coremain.Run()
}

