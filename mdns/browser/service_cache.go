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

func (sc *serviceCache) getExpiry(instance string) time.Time {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	if tracked, ok := (*sc.services)[instance]; ok {
		return tracked.expiry
	}
	return time.Time{} // Zero time if not found
}
