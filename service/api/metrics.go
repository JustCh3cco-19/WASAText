package api

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/julienschmidt/httprouter"
)

type requestMetrics struct {
	startedAt time.Time
	total     atomic.Uint64
	errors    atomic.Uint64
	inFlight  atomic.Int64
}

func newRequestMetrics() *requestMetrics { return &requestMetrics{startedAt: time.Now()} }

func (m *requestMetrics) begin() { m.inFlight.Add(1) }
func (m *requestMetrics) observe(status int) {
	m.inFlight.Add(-1)
	m.total.Add(1)
	if status >= http.StatusInternalServerError {
		m.errors.Add(1)
	}
}

func (rt *_router) metrics(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w,
		"# TYPE wasatext_http_requests_total counter\nwasatext_http_requests_total %d\n"+
			"# TYPE wasatext_http_errors_total counter\nwasatext_http_errors_total %d\n"+
			"# TYPE wasatext_http_requests_in_flight gauge\nwasatext_http_requests_in_flight %d\n"+
			"# TYPE wasatext_uptime_seconds gauge\nwasatext_uptime_seconds %.0f\n",
		rt.metricsState.total.Load(), rt.metricsState.errors.Load(), rt.metricsState.inFlight.Load(), time.Since(rt.metricsState.startedAt).Seconds())
}
