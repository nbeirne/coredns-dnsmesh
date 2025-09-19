
package browser

import (
	"context"

	"github.com/grandcat/zeroconf"
)


// allow for mocking in tests
type ZeroconfInterface interface {
	NewResolver(...zeroconf.ClientOption) (ResolverInterface, error)
}

type ZeroconfImpl struct{}

func (z ZeroconfImpl) NewResolver(opts ...zeroconf.ClientOption) (ResolverInterface, error) {
	return zeroconf.NewResolver(opts...)
}

type ResolverInterface interface {
	Browse(ctx context.Context, service string, domain string, entries chan<- *zeroconf.ServiceEntry) error
	Lookup(ctx context.Context, instance string, service string, domain string, entries chan<- *zeroconf.ServiceEntry) error
}

