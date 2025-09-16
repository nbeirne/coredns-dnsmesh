

## Useful Commands
```
# query a service @ a domain
dig @224.0.0.251 -p 5353 -t ptr _airplay._tcp.local


# find a host
dig -p 5353 @224.0.0.251 bowie.local
```


## TODO

### Now

- NewMeshDnsProvider 

- better strategy than a 2s browse loop
    - browse - cache results until a TTL is hit. At that point restart the browse.
    - browsing - maintaining list when given via chan
    - browsing - removal when TTL is 0


### Maybe or Later

- Deconflicting the advertiser (ie, UUIDs or adding bracketed numbering after discovery)

- TCP support (ie, not dns over udp)
- better security w.r.t. which servers may respond to which domains.
- browsing: also look at port to filter out 'self' instances?? make this configurable?

