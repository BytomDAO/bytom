package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/bytom/metrics"
)

var (
	latencyMu sync.Mutex
	latencies = map[string]*metrics.RotatingLatency{}

	latencyRange = map[string]time.Duration{
		crosscoreRPCPrefix + "get-block":         20 * time.Second,
		crosscoreRPCPrefix + "signer/sign-block": 5 * time.Second,
		crosscoreRPCPrefix + "get-snapshot":      30 * time.Second,
		// the rest have a default range
	}
)

// latency returns a rotating latency histogram for the given request.
func latency(tab *http.ServeMux, req *http.Request) *metrics.RotatingLatency {
	latencyMu.Lock()
	defer latencyMu.Unlock()
	if l := latencies[req.URL.Path]; l != nil {
		return l
	}
	// Create a histogram only if the path is legit.
	if _, pat := tab.Handler(req); pat == req.URL.Path {
		d, ok := latencyRange[req.URL.Path]
		if !ok {
			d = 100 * time.Millisecond
		}
		l := metrics.NewRotatingLatency(5, d)
		latencies[req.URL.Path] = l
		metrics.PublishLatency(req.URL.Path, l)
		return l
	}
	return nil
}
