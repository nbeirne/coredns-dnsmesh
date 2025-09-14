# build from docker/ dir: docker build --build-context "src=.." .

FROM golang:1.24.2-alpine3.21 AS build

WORKDIR /go/src/coredns

# pre-fetch all deps before copying src files
RUN apk add git make bind-tools && \
    git clone --depth 1 --branch=v1.11.3 https://github.com/coredns/coredns /go/src/coredns  && \
    git clone --branch=main https://github.com/nbeirne/coredns-mdns/ /go/src/coredns/plugin/mdns && \
    git clone --branch=main https://github.com/scottt732/coredns-traefik /go/src/coredns/plugin/traefik && \
    git clone --branch=main https://github.com/damomurf/coredns-tailscale /go/src/coredns/plugin/tailscale && \
    git clone --branch=master https://github.com/tmeckel/coredns-finalizer /go/src/coredns/plugin/finalize && \
    rm -rf /go/src/coredns/plugin/tailscale/go.sum /go/src/coredns/plugin/tailscale/go.mod && \
    rm -rf /go/src/coredns/plugin/finalize/go.sum /go/src/coredns/plugin/finalize/go.mod && \
    go get github.com/networkservicemesh/fanout@e635d0cda3c4411abc0a1e1ca2bb38a3af04be44 && \
    sed -i s/forward:forward/finalize:finalize\\ntraefik:traefik\\ntailscale:tailscale\\nforward:forward/ /go/src/coredns/plugin.cfg && \
    make check && go build

COPY --from=src . /go/src/coredns-dnsmesh
RUN echo "installing..." && \
    cp -r /go/src/coredns-dnsmesh/test_provider /go/src/coredns/plugin/dnsmesh_test_provider && \
    rm /go/src/coredns/plugin/dnsmesh_test_provider/go.mod /go/src/coredns/plugin/dnsmesh_test_provider/go.sum && \
    sed -i s/forward:forward/dnsmesh_test_provider:dnsmesh_test_provider\\nforward:forward/ /go/src/coredns/plugin.cfg && \
    echo "done installing..." && \
    cp -r /go/src/coredns-dnsmesh/mdns /go/src/coredns/plugin/dnsmesh_mdns && \
    rm /go/src/coredns/plugin/dnsmesh_mdns/go.mod /go/src/coredns/plugin/dnsmesh_mdns/go.sum && \
    sed -i s/forward:forward/dnsmesh_mdns:dnsmesh_mdns\\nforward:forward/ /go/src/coredns/plugin.cfg && \
    echo "done installing..." && \
    go mod edit -replace github.com/nbeirne/coredns-dnsmesh/util=/go/src/coredns-dnsmesh/util && \
    make check && go build

FROM alpine:3.21.3
RUN apk add --no-cache ca-certificates bind-tools curl

RUN ls
COPY --from=build /go/src/coredns/coredns /
COPY --from=src docker/run.sh /

ENTRYPOINT ["/run.sh"]

