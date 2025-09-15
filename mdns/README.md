

## Useful Commands
```
# query a service @ a domain
dig @224.0.0.251 -p 5353 -t ptr _airplay._tcp.local


# find a host
dig -p 5353 @224.0.0.251 bowie.local
```


## TODO

- Figure out fanout settings **now**
- simplify dns provider **now**

- lots of testing **now**
    - filtering in the provider
    - browsing - maintaining list when given via chan
    - browsing - removal when TTL is 0
    - advertising - deconfliction

- try out query and advertising non-local domains

- TCP support (ie, not dns over udp)
- better security w.r.t. which servers may respond to which domains.


### Advertising
- Deconflicting the advertiser (ie, UUIDs or adding bracketed numbering after discovery)


### Browsing
- better strategy than a 2s browse loop
    - browse - cache results until a TTL is hit. At that point restart the browse.
- remove item when TTL is 0
- also look at port to filter out 'self' instances?? make this configurable?
- fanout strategy
- NewMeshDnsProvider


### Setup and Config

