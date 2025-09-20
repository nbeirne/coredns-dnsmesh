package main

import (
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/debug"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/reload"
	_ "github.com/coredns/records"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"


	_ "github.com/nbeirne/coredns-dnsmesh/self"
)

var directives = []string{
	"reload",
	"debug",
	"errors",
	"log",
	"self",
	"whoami",
}

func init() {
	dnsserver.Directives = directives
}

func main() {
	coremain.Run()
}

