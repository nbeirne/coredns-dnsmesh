### Project Overview

This project provides a CoreDNS plugin, `dnsmesh`, which enables dynamic, zero-configuration DNS service discovery and resolution within a local network. It consists of two main components:

1.  **`dnsmesh_mdns_forward`**: A forwarding plugin that discovers other DNS servers on the network via mDNS/Zeroconf. It then forwards (fans out) DNS queries for a specified zone to these discovered peers.
2.  **`dnsmesh_mdns_advertise`**: An advertising plugin that announces the presence of the CoreDNS server itself via mDNS, allowing other `dnsmesh_mdns_forward` instances to discover it.

Together, these plugins allow you to create a resilient and self-organizing "mesh" of DNS servers. This is ideal for environments like home labs, small offices, or containerized setups where services and their IP addresses may change frequently.

### Features

*   **Automatic Peer Discovery**: No static configuration of upstream DNS servers is needed. Peers are discovered automatically as they join the network.
*   **Resilient Forwarding**: DNS queries are fanned out to all discovered peers, providing resilience if one or more peers become unavailable.
*   **Configurable Service Types**: Discover and advertise any mDNS service type (e.g., `_dns._udp`), allowing you to create separate meshes for different purposes.
*   **Advanced Filtering**: Control which discovered services are used as upstreams based on instance name, IP address family (IPv4/IPv6), and network interface.
*   **Graceful Operation**: The mDNS browser handles service TTLs, refreshes, and expirations to maintain an up-to-date list of active peers.
*   **Easy Integration**: Plugs into CoreDNS and is configured via the `Corefile`.

### How It Works

1.  You run CoreDNS on multiple machines on the same local network.
2.  On each machine, the `Corefile` is configured with the `dnsmesh_mdns_advertise` plugin. This makes each CoreDNS instance broadcast its availability as a specific mDNS service (e.g., `_dns._udp`).
3.  The `Corefile` is also configured with the `dnsmesh_mdns_forward` plugin for a specific zone (e.g., `mesh.local`). This plugin continuously browses for the same mDNS service (`_dns-mesh._tcp.local`).
4.  When a DNS query for `some-service.mesh.local` arrives, the `dnsmesh_mdns_forward` plugin forwards the query to all the peer CoreDNS instances it has discovered on the network.
5.  The first peer to respond with a successful answer provides the resolution.

This creates a decentralized system where any node in the mesh can resolve names managed by any other node.

### Building CoreDNS with `dnsmesh`

To use this plugin, you need to compile a custom version of CoreDNS.

1.  Clone the CoreDNS repository:
    ```sh
    git clone https://github.com/coredns/coredns.git
    cd coredns
    ```

2.  Add the `dnsmesh` plugin to `plugin.cfg`. The entry should point to the Go module path of your plugin:
    ```
    # Add this line to plugin.cfg, adjusting the module path as needed
    dnsmesh_mdns_advertise:github.com/nbeirne/coredns-dnsmesh/mdns
    dnsmesh_mdns_forward:github.com/nbeirne/coredns-dnsmesh/mdns
    ```

3.  Fetch the dependencies and generate the CoreDNS source files:
    ```sh
    go get github.com/nbeirne/coredns-dnsmesh/mdns
    go generate
    ```

4.  Build CoreDNS:
    ```sh
    go build
    ```

You will now have a `coredns` executable in your directory that includes the `dnsmesh` plugin.

### Configuration

You configure the plugins in your `Corefile`. Here is a complete example demonstrating a typical setup.

```
# Corefile
. {
    # Standard plugins
    log
    errors
    cache 30

    # Advertise this CoreDNS instance as part of the mesh
    # It will be discoverable by other nodes.
    dnsmesh_mdns_advertise {
        # The service type for the mesh. Must match the query plugin.
        type _dns._udp
        # A unique name for this instance. Defaults to the hostname.
        instance_name node-1
        # The TTL for the mDNS announcement.
        ttl 120
    }
}

# Define a zone that will be resolved by the mesh.
mesh.local:53 {
    log

    # The query plugin for the mesh zone.
    dnsmesh_mdns_forward mesh.local {
        # The service type to discover. Must match the advertise plugin.
        type _dns._udp
        # Filter out this server's own advertisement to prevent query loops.
        ignore_self true
        # How long to wait for a response from peers.
        timeout 2s
    }

    # Fallback for names not found in the mesh
    forward . /etc/resolv.conf
}
```

#### `dnsmesh_mdns_advertise` Options

*   **`instance_name <name>`**: Sets the instance name for the mDNS advertisement. Defaults to the machine's hostname.
*   **`type <service>`**: The mDNS service type to advertise. Defaults to `_dns._udp`.
*   **`port <port>`**: The port to advertise. Defaults to the port CoreDNS is listening on.
*   **`ttl <seconds>`**: The Time-To-Live for the mDNS record in seconds. Defaults to `320`.
*   **`iface_bind_subnet <cidr>`**: Binds the advertisement to the network interface associated with the given subnet (e.g., `192.168.1.0/24`).

#### `dnsmesh_mdns_query` Options

The first argument to `dnsmesh_mdns_query` is the zone it is responsible for.

*   **`type <service>`**: The mDNS service type to browse for. Defaults to `_dns._udp`.
*   **`ignore_self <true|false>`**: If `true`, ignores discovered services running on the same machine to prevent query loops. Defaults to `false`.
*   **`filter <regex>`**: A regular expression to filter discovered services by their instance name. Only matching instances will be used as upstreams.
*   **`address_mode <mode>`**: Defines the IP address preference when multiple are available for a service. Modes are:
    *   `prefer_ipv6` (default)
    *   `prefer_ipv4`
    *   `only_ipv6`
    *   `only_ipv4`
*   **`addresses_per_host <count>`**: Limits the number of IP addresses to use per discovered host. Defaults to `0` (unlimited).
*   **`iface_bind_subnet <cidr>`**: Restricts browsing to the network interface associated with the given subnet.
*   **`timeout <duration>`**: The overall timeout for a fanned-out request (e.g., `500ms`, `2s`). Defaults to `2s`.
*   **`attempts <count>`**: The number of times to try each discovered upstream server if a query fails. Defaults to `1`.
*   **`worker_count <count>`**: The number of parallel queries to run. Defaults to `10`.
