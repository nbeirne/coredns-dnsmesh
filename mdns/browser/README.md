# mDNS/Zeroconf Service Browser

This Go package provides a robust, long-running mDNS/Zeroconf service browser. It is designed to be integrated into applications like CoreDNS plugins that need to maintain an up-to-date list of services advertised on the local network.

The browser continuously listens for mDNS announcements, manages a cache of discovered services, and handles service lifecycle events such as expiry and explicit "goodbye" messages.

## Features

- **Continuous Browsing**: Runs a continuous mDNS `Browse` operation to discover services.
- **Service Caching**: Maintains an in-memory cache of discovered services.
- **Proactive Refresh**: When a service's Time-To-Live (TTL) is nearing expiration (at 80% of its original duration), the browser performs a targeted `Lookup` to refresh its details and prevent it from expiring from the cache unnecessarily.
- **Graceful Expiry**: Services are automatically removed from the cache when their TTL expires or when a "goodbye" announcement (TTL=0) is received.
- **Concurrency Safe**: Designed for concurrent use with proper synchronization.
- **Clean Shutdown**: Ensures all background goroutines are cleanly terminated when the browser is stopped.
- **Interface Selection**: Can be configured to browse on all network interfaces or a specific subset.
- **Testable**: Key components are abstracted behind interfaces (`ZeroconfInterface`, `ResolverInterface`) to allow for comprehensive mocking and testing.

## How It Works

1.  **Start**: The `Start()` method kicks off a `browseLoop` in a background goroutine.
2.  **Browse**: The `browseLoop` creates a `ZeroconfSession` and begins browsing for the specified service type (e.g., `_my-service._tcp`).
3.  **Process Entries**: Discovered service entries are sent over a channel to `processEntries`.
    - If an entry has a `TTL` of 0, it is immediately removed from the cache.
    - If an entry has a positive `TTL`, it is added or updated in the service cache.
4.  **TTL Management**: When a service is discovered or updated, a timer is set.
    - The timer is scheduled to fire when 80% of the service's TTL has elapsed.
    - When the timer fires, a `Lookup` is performed for that specific service instance to get a refreshed `ServiceEntry` with a new TTL.
    - If the `Lookup` fails, the service is removed from the cache.
5.  **Service Access**: The `Services()` method provides a snapshot of the currently valid (non-expired) services in the cache.
6.  **Stop**: The `Stop()` method cancels the main context, which gracefully shuts down the browsing session and all associated goroutines. It waits for all routines to exit before returning.

## Usage

### As a Library

You can integrate the `ZeroconfBrowser` into your application.

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/nbeirne/coredns-dnsmesh/mdns/browser"
)

func main() {
	// Create a new browser for the "_http._tcp" service on the "local." domain.
	// Pass nil for interfaces to scan on all available interfaces.
	b := browser.NewZeroconfBrowser("local.", "_http._tcp", nil)

	// Start the browser. This runs in the background.
	if err := b.Start(); err != nil {
		panic(err)
	}
	defer b.Stop()

	fmt.Println("Browsing for services. Press Ctrl+C to exit.")

	// Periodically print the discovered services.
	go func() {
		for {
			time.Sleep(5 * time.Second)
			services := b.Services()
			fmt.Printf("\n--- Found %d services ---\n", len(services))
			for _, s := range services {
				fmt.Printf("Instance: %s, Host: %s, Addr: %v, Port: %d\n", s.Instance, s.HostName, s.AddrIPv4, s.Port)
			}
		}
	}()

	// Wait for a SIGINT (Ctrl-C)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	fmt.Println("\nShutting down...")
}
```

### Standalone CLI

A simple command-line utility is provided in the `cmd/` directory to demonstrate the browser's functionality.

**Build and Run:**

```sh
cd cmd
go build
./main --service=_http._tcp
```
