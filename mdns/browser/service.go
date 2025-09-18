package browser

import (
	"time"

	"github.com/grandcat/zeroconf"
)

type trackedService struct {
	entry       *zeroconf.ServiceEntry
	originalTTL time.Duration
	expiry      time.Time
}

func (m *ZeroconfBrowser) calculateNextRefresh() time.Duration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.calculateNextRefresh_nolock()
}

func (m *ZeroconfBrowser) calculateNextRefresh_nolock() time.Duration {
	// Default refresh interval if no services are found.
	const defaultRefresh = 60 * time.Second
	// Minimum refresh to avoid busy-looping on expired entries.
	const minRefresh = 2 * time.Second

	if len(*m.services) == 0 {
		return defaultRefresh
	}

	now := time.Now()
	var nextRefresh time.Time

	for _, s := range *m.services {
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

func (m *ZeroconfBrowser) removeExpiredServices() {
	log.Debugf("Removing expired services...")

	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for instance, s := range *m.services {
		if now.After(s.expiry) {
			log.Infof("TTL expired for service, removing: %s", instance)
			delete(*m.services, instance)
		}
	}
}

// addEntry receives an entry and adds it to the service map or removes it if TTL is 0.
func (m *ZeroconfBrowser) addEntry(entry *zeroconf.ServiceEntry) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if entry.TTL == 0 {
		log.Infof("Service expired via TTL=0: %s", entry.Instance)
		delete(*m.services, entry.Instance)
		return
	}

	tracked := &trackedService{
		entry:       entry,
		originalTTL: time.Duration(entry.TTL) * time.Second,
		expiry:      time.Now().Add(time.Duration(entry.TTL) * time.Second),
	}

	log.Infof("Service Instance: %s\n    HostName: %s\n    AddrIPv4: %s\n    AddrIPv6: %s\n    Port: %d\n    TTL: %d\n", entry.Instance, entry.HostName, entry.AddrIPv4, entry.AddrIPv6, entry.Port, entry.TTL)
	(*m.services)[entry.Instance] = tracked
}
