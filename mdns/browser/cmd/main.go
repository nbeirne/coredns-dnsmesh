package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/nbeirne/coredns-dnsmesh/mdns/browser"
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
	ifaceStrs := arrayFlags{}

	service := flag.String("service", "", "The mDNS service to browse for (required)")
	logLevel := flag.String("log-level", "info", "Set log level: debug, info, warn, error")
	flag.Var(&ifaceStrs, "iface", "The interface to scan (can be specified multiple times)")

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
	b.Log = NewPrintLogger(*logLevel)
	b.Start()

	// Wait for a SIGINT (Ctrl-C)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	fmt.Println("\nShutting down...")
	b.Stop()
	fmt.Println("Browser test finished.")
}

const (
	levelDebug = iota
	levelInfo
	levelWarn
	levelError
)

type PrintLogger struct {
	level int
}

func NewPrintLogger(levelStr string) *PrintLogger {
	level := levelInfo // default
	switch strings.ToLower(levelStr) {
	case "debug":
		level = levelDebug
	case "info":
		level = levelInfo
	case "warn":
		level = levelWarn
	case "error":
		level = levelError
	}
	return &PrintLogger{level: level}
}

func (l *PrintLogger) Debugf(format string, args ...interface{}) {
	if l.level <= levelDebug {
		log.Printf("DEBUG: "+format, args...)
	}
}

func (l *PrintLogger) Infof(format string, args ...interface{}) {
	if l.level <= levelInfo {
		log.Printf("INFO: "+format, args...)
	}
}

func (l *PrintLogger) Warningf(format string, args ...interface{}) {
	if l.level <= levelWarn {
		log.Printf("WARN: "+format, args...)
	}
}

func (l *PrintLogger) Errorf(format string, args ...interface{}) {
	if l.level <= levelError {
		log.Printf("ERROR: "+format, args...)
	}
}
