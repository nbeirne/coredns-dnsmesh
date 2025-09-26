package browser

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// ServiceRefresher manages the lifecycle of a single discovered mDNS service.
// It is responsible for scheduling and performing proactive lookups to refresh
// the service's TTL before it expires.
type ServiceRefresher struct {
	Log Logger

	service   string
	domain    string
	session   ResolverInterface
	cache     *serviceCache
	entriesCh chan<- *zeroconf.ServiceEntry
	onRemove  func(instance *zeroconf.ServiceEntry)

	timers      map[string]*time.Timer
	timersMutex sync.Mutex
}

func newServiceRefresher(service, domain string, session ResolverInterface, cache *serviceCache, entriesCh chan<- *zeroconf.ServiceEntry, onRemove func(instance *zeroconf.ServiceEntry)) *ServiceRefresher {
	return &ServiceRefresher{
		Log:       NoLogger{},
		service:   service,
		domain:    domain,
		session:   session,
		cache:     cache,
		entriesCh: entriesCh,
		onRemove:  onRemove,
		timers:    make(map[string]*time.Timer),
	}
}

// Refresh schedules a proactive lookup for the given service entry.
func (r *ServiceRefresher) Refresh(ctx context.Context, entry *zeroconf.ServiceEntry) {
	// Calculate the base refresh time (e.g., 80% of TTL).
	baseRefreshSeconds := float64(entry.TTL) * TTLRefreshThreshold
	// Calculate jitter as a random percentage of the base refresh time.
	jitter := (rand.Float64()*2 - 1) * JitterFactor * baseRefreshSeconds
	// Apply jitter to the base refresh time.
	refreshDuration := time.Duration((baseRefreshSeconds + jitter) * float64(time.Second))

	r.Log.Debugf("Scheduling refresh for '%s' in %v", entry.Instance, refreshDuration)

	r.timersMutex.Lock()
	defer r.timersMutex.Unlock()

	// Stop any existing timer for this service instance.
	if timer, ok := r.timers[entry.Instance]; ok {
		timer.Stop()
	}

	// Create a new timer to trigger a lookup for this service.
	r.timers[entry.Instance] = time.AfterFunc(refreshDuration, func() {
		originalExpiry := r.cache.getExpiry(entry.Instance)

		lookupTimeout := (time.Duration(entry.TTL) * time.Second) - refreshDuration
		lookupTimeout = min(lookupTimeout, MaxLookupTimeout)
		r.Log.Infof("TTL for %v is low, performing lookup with timeout %v", entry.Instance, lookupTimeout)
		lCtx, lCancel := context.WithTimeout(ctx, lookupTimeout)
		defer lCancel()

		err := r.session.Lookup(lCtx, entry.Instance, r.service, r.domain, r.entriesCh)

		currentExpiry := r.cache.getExpiry(entry.Instance)
		if err != nil || !currentExpiry.After(originalExpiry) {
			r.Log.Warningf("Lookup for service '%s' failed or did not refresh. Triggering a general browse query as a fallback.", entry.Instance)
			// Fallback: if the targeted lookup fails, trigger a general browse query.
			// This gives the slow host another chance to be discovered.
			bCtx, bCancel := context.WithTimeout(ctx, 5*time.Second) // Short-lived browse
			defer bCancel()
			_ = r.session.Browse(bCtx, r.service, r.domain, r.entriesCh)
		} else {
			r.Log.Debugf("Lookup complete for %s...", entry.Instance)
		}
	})
}

// StopAll cancels all active refresh timers.
func (r *ServiceRefresher) StopAll() {
	r.timersMutex.Lock()
	defer r.timersMutex.Unlock()
	for _, timer := range r.timers {
		timer.Stop()
	}
}
