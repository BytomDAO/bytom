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
		d := 100 * time.Millisecond
		l := metrics.NewRotatingLatency(5, d)
		latencies[req.URL.Path] = l
		metrics.PublishLatency(req.URL.Path, l)
		return l
	}
	return nil
}
