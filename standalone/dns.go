package main

import (
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/view"
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/whoami"
	//_ "github.com/damomurf/coredns-tailscale"
	_ "github.com/kevinjqiu/coredns-dockerdiscovery"
	_ "github.com/openshift/coredns-mdns"
	_ "github.com/nbeirne/coredns-traefik"
	_ "github.com/tmeckel/coredns-finalizer"

	"github.com/coredns/coredns/core/dnsserver"
	_ "github.com/coredns/coredns/plugin/file"
	_ "github.com/coredns/coredns/plugin/auto"
	_ "github.com/coredns/coredns/plugin/rewrite"
	_ "github.com/coredns/coredns/plugin/reload"
	_ "github.com/coredns/coredns/plugin/ready"
	_ "github.com/coredns/coredns/plugin/root"
	_ "github.com/coredns/records"
	"github.com/coredns/coredns/coremain"

	_ "github.com/networkservicemesh/fanout"

	_ "github.com/nbeirne/coredns-dnsmesh/test_provider"
	_ "github.com/nbeirne/coredns-dnsmesh/mdns"
	_ "github.com/nbeirne/coredns-dnsmesh/util"
)

var directives = []string{
	//"root",
	// metadata:metadata
	// geoip:geoip
	// cancel:cancel
	// tls:tls
	// timeouts:timeouts
	"reload",
	// nsid:nsid
	// bufsize:bufsize
	// bind:bind
	"debug",
	//"trace",
	"ready",
	"health:health",
	// pprof:pprof
	// prometheus:metrics
	"errors",
	"log",
	// dnstap:dnstap
	// local:local
	// dns64:dns64
	// acl:acl
	// any:any
	// chaos:chaos
	// loadbalance:loadbalance
	// tsig:tsig
	"cache",
	"finalize",
	"rewrite",
	// header:header
	// dnssec:dnssec
	// autopath:autopath
	// minimal:minimal
	// template:template
	// transfer:transfer
	"hosts",
	//"auto",
	//"records",
	//"file",
	// route53:route53
	// azure:azure
	// clouddns:clouddns
	// k8s_external:k8s_external
	// kubernetes:kubernetes
	//"mdns",
	//"tailscale", // tailscale before traefik to handle traefik cname
	"traefik",
	//"docker",
	// secondary:secondary
	// etcd:etcd
	// loop:loop
	"dnsmesh_test_provider",
	"dnsmesh_mdns",
	"dnsmesh_mdns_advertise",
	"forward",
	"fanout",
	// grpc:grpc
	// erratic:erratic
	"whoami",
	// on:github.com/coredns/caddy/onevent
	// sign:sign
	"view",
}

func init() {
	dnsserver.Directives = directives
}

func main() {
	coremain.Run()
}

