module github.com/FrazerSeymour/dns

go 1.25.1

require (
	github.com/coredns/coredns v1.12.4
	github.com/coredns/records v0.0.0-20230310133434-a3157e710d9e
	github.com/kevinjqiu/coredns-dockerdiscovery v0.0.0-20240325134730-2f65ec48a254
	github.com/nbeirne/coredns-dnsmesh/mdns v0.0.0-00010101000000-000000000000
	github.com/nbeirne/coredns-dnsmesh/test_provider v0.0.0-00010101000000-000000000000
	github.com/nbeirne/coredns-dnsmesh/util v0.0.0-00010101000000-000000000000
	github.com/nbeirne/coredns-traefik v0.0.0-20241209160058-8a0511be5456
	github.com/networkservicemesh/fanout v1.11.4-0.20250612154940-e635d0cda3c4
	github.com/openshift/coredns-mdns v0.0.0-20210625150643-8c0b6474833f
	github.com/tmeckel/coredns-finalizer v0.0.0-20250905220621-34ea159d800b
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/celebdor/zeroconf v0.0.0-20210412110229-8ba34664402f // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/containerd v1.7.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/coredns/caddy v1.1.2-0.20241029205200-8de985351a98 // indirect
	github.com/dnstap/golang-dnstap v0.4.0 // indirect
	github.com/docker/docker v25.0.5+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/expr-lang/expr v1.17.6 // indirect
	github.com/farsightsec/golang-framestream v0.3.0 // indirect
	github.com/flynn/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/fsouza/go-dockerclient v1.9.7 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grandcat/zeroconf v1.0.0 // indirect
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/miekg/dns v1.1.68 // indirect
	github.com/moby/patternmatcher v0.5.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.2.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nbeirne/coredns-dnsmesh/mdns/browser v0.0.0-00010101000000-000000000000 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/openshift/mdns-publisher v0.0.0-20220222182051-8fef1ccb075f // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.23.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	golang.org/x/tools v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/nbeirne/coredns-dnsmesh/util => ../util

replace github.com/nbeirne/coredns-dnsmesh/mdns => ../mdns

replace github.com/nbeirne/coredns-dnsmesh/test_provider => ../test_provider

replace github.com/nbeirne/coredns-dnsmesh/mdns/browser => ../mdns/browser
