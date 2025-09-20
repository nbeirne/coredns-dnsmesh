package browser

import (
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

type trackedService struct {
	entry       *zeroconf.ServiceEntry
	originalTTL time.Duration
	expiry      time.Time
}

type serviceCache struct {
	mutex    *sync.RWMutex
	services *map[string]*trackedService
}

func newServiceCache() *serviceCache {
	services := make(map[string]*trackedService)
	return &serviceCache{
		mutex:    &sync.RWMutex{},
		services: &services,
	}
}

func (sc *serviceCache) calculateNextRefresh() time.Duration {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	return sc.calculateNextRefresh_nolock()
}

func (sc *serviceCache) calculateNextRefresh_nolock() time.Duration {
	// Default refresh interval if no services are found.
	const defaultRefresh = 60 * time.Second
	// Minimum refresh to avoid busy-looping on expired entries.
	const minRefresh = 2 * time.Second

	if len(*sc.services) == 0 {
		return defaultRefresh
	}

	now := time.Now()
	var nextRefresh time.Time

	for _, s := range *sc.services {
		// Refresh when 80% of the TTL has passed (i.e., 20% remains).
		// The refresh time is the service's expiry time minus 20% of its original TTL.
		refreshTime := s.expiry.Add(-(s.originalTTL * 2 / 10))

		log.Debugf("CALC REF %v %v %v", s.expiry, s.originalTTL, refreshTime)

		// If the calculated refresh time is already in the past, we should refresh very soon.
		if refreshTime.Before(now) {
			return minRefresh
		}

		if nextRefresh.IsZero() || refreshTime.Before(nextRefresh) {
			nextRefresh = refreshTime
		}
	}

	return time.Until(nextRefresh)
}

func (sc *serviceCache) removeExpiredServices() {
	log.Debugf("Removing expired services...")

	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	now := time.Now()
	for instance, s := range *sc.services {
		if now.After(s.expiry) {
			log.Infof("TTL expired for service, removing: %s", instance)
			delete(*sc.services, instance)
		}
	}
}

func (sc *serviceCache) removeEntry(instance string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	delete(*sc.services, instance)
}

// addEntry receives an entry and adds it to the service map or removes it if TTL is 0.
func (sc *serviceCache) addEntry(entry *zeroconf.ServiceEntry) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	tracked := &trackedService{
		entry:       entry,
		originalTTL: time.Duration(entry.TTL) * time.Second,
		expiry:      time.Now().Add(time.Duration(entry.TTL) * time.Second),
	}

	log.Infof("Service Instance: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d\n", entry.Instance, entry.HostName, entry.AddrIPv4, entry.AddrIPv6, entry.Port, entry.TTL)
	(*sc.services)[entry.Instance] = tracked
}

func (sc *serviceCache) getServices() []*zeroconf.ServiceEntry {
	now := time.Now()
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	serviceEntries := make([]*zeroconf.ServiceEntry, 0, len(*sc.services))
	for _, s := range *sc.services {
		if now.After(s.expiry) {
			continue
		}
		serviceEntries = append(serviceEntries, s.entry)
	}
	return serviceEntries
}
