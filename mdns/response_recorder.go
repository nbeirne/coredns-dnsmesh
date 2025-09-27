package mdns

import (
	"github.com/miekg/dns"
)

// responseRecorder is a dns.ResponseWriter that records the status code of the response.
type responseRecorder struct {
	dns.ResponseWriter
	Rcode int
	Msg   *dns.Msg
}

// NewResponseRecorder returns a new responseRecorder.
func NewResponseRecorder(w dns.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, Rcode: -1}

}

// WriteMsg records the status code and calls the underlying ResponseWriter's WriteMsg method.
func (r *responseRecorder) WriteMsg(res *dns.Msg) error {
	r.Rcode = res.Rcode
	r.Msg = res
	return r.ResponseWriter.WriteMsg(res)
}

// Hijack implements dns.ResponseWriter.
func (r *responseRecorder) Hijack() {
	r.Hijack()
}

// Close implements dns.ResponseWriter.
func (r *responseRecorder) Close() error {
	return r.ResponseWriter.Close()
}
