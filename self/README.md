# CoreDNS Self Plugin

## Description

The `self` plugin returns the IP address of the server when a query is made for a specific hostname. It returns the IP of the interface that received the request.

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
