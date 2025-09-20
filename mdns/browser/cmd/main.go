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

type arrayFlags []string

// String is an implementation of the flag.Value interface
func (i *arrayFlags) String() string {
    return fmt.Sprintf("%v", *i)
}

// Set is an implementation of the flag.Value interface
func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}


func main() {
	clog.D.Set()

	ifaceStrs := arrayFlags{}

	service := flag.String("service", "", "The mDNS service to browse for (required)")
	flag.Var(&ifaceStrs, "", "The interface to scan")

	flag.Parse()

	if *service == "" {
		fmt.Fprintln(os.Stderr, "Error: the --service flag is required.")
		os.Exit(1)
	}

	ifaces := []net.Interface{}
	for _, ifaceStr := range ifaceStrs {
		iface, err := net.InterfaceByName(ifaceStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No interface found with name %s", iface)
			os.Exit(1)
		}
		ifaces = append(ifaces, *iface)
	}

	b := browser.NewZeroconfBrowser("local.", *service, &ifaces)
	b.Start()

	// Wait for a SIGINT (Ctrl-C)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	fmt.Println("\nShutting down...")
	b.Stop()
	fmt.Println("Browser test finished.")
}
