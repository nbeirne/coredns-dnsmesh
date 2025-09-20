# Name

*self* - returns the server's own IP address based on the client's subnet.

## Description

The *self* plugin provides a split-horizon DNS capability by responding to queries with the IP address(es) of the server from the perspective of the client's network. When a query is received for a name within one of the configured zones, *self* identifies which of the server's network interfaces is on the same subnet as the client. It then responds with the IP address(es) from that interface, filtering for A (IPv4) or AAAA (IPv6) records based on the query type.

This is useful in environments where a server has multiple IP addresses on different networks, and clients need to connect to the address that is local to them.

If no matching interface is found for the client's source IP, or if a matching interface has no IPs of the requested type (e.g., an AAAA query for an IPv4-only interface), the plugin returns an `NXDOMAIN` response.

## Syntax

```
self [ZONES...]
```

* `ZONES` - a list of zones for which the plugin should be active.

## Examples

```
. {
    self example.org
}
```
