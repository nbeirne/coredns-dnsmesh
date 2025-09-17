package main

import (
	"fmt"
	"flag"
	"net"
	"os"
	"os/signal"

	"github.com/nbeirne/coredns-dnsmesh/mdns/browser"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

func main() {
	service := flag.String("service", "", "The mDNS service to browse for (required)")
	subnetCIDR := flag.String("subnet", "", "The subnet to scan in CIDR notation (optional)")
	flag.Parse()

	if *service == "" {
		fmt.Fprintln(os.Stderr, "Error: the --service flag is required.")
		os.Exit(1)
	}

	clog.D.Set()

	var subnet *net.IPNet
	if *subnetCIDR != "" {
		var err error
		_, subnet, err = net.ParseCIDR(*subnetCIDR)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing subnet '%s': %v\n", *subnetCIDR, err)
			os.Exit(1)
		}
		fmt.Printf("Browsing for mDNS service '%s' on subnet '%s'. Press Ctrl-C to exit.\n", *service, *subnetCIDR)
	} else {
		fmt.Printf("Browsing for mDNS service '%s' on all interfaces. Press Ctrl-C to exit.\n", *service)
	}

	b := browser.NewMdnsBrowser("local.", *service, subnet)
	b.Start()

	// Wait for a SIGINT (Ctrl-C)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	fmt.Println("\nShutting down...")
	b.Stop()
	fmt.Println("Browser test finished.")
}
