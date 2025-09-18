package browser

// Requirements of this mDNS Browser:
// 1. It should have a long running zeroconf..NewResolver().Browse process running.
// 2. When a new entry is received, it should be tracked in MdnsBrowser.services.
// 3. When an entry with TTL = 0 is received, it should be removed from MdnsBrowser.services.
// 4. When an entries TTL reaches 20% of its original value, the Browse() process should be canceled and restarted.
// 5. When an entries TTL reaches 0, it should be removed from MdnsBrowser.services.
// 6. The stop function should wait until all go routines are finished (especially the browseLoop and receiveEntries routines)

import (
	"github.com/grandcat/zeroconf"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("mdns_browser")

type MdnsBrowserInterface interface {
	Start() error
	Stop()
	Services() []*zeroconf.ServiceEntry
}
