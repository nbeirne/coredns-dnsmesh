# CoreDNS Corefile
#
# This configuration will respond to DNS queries for specific hostnames
# with the IP address of the server interface that received the query.

# Define a server block for the "whoami" service.
# This will handle queries for whoami.example.com and whoami-ipv6.example.com.
whoami.example.com whoami-ipv6.example.com {
    # The whoami plugin responds with the IP address of the interface
    # that received the request. It correctly handles both IPv4 and IPv6.
    whoami
    log
}

# This is the default server block for all other queries.
. {
    # Log all requests to standard output. Useful for debugging.
    log

    # Handle health checks for orchestrators like Kubernetes.
    # Responds to "health.coredns" with "OK".
    health
}
