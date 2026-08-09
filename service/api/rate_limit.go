package api

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

const (
	authAttemptsPerWindow = 10
	authWindow            = time.Minute
)

type rateEntry struct {
	count   int
	resetAt time.Time
}
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
}

func newRateLimiter() *rateLimiter { return &rateLimiter{entries: make(map[string]rateEntry)} }

func (limiter *rateLimiter) allow(key string, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	entry := limiter.entries[key]
	if entry.resetAt.IsZero() || !now.Before(entry.resetAt) {
		limiter.entries[key] = rateEntry{count: 1, resetAt: now.Add(authWindow)}
		return true
	}
	if entry.count >= authAttemptsPerWindow {
		return false
	}
	entry.count++
	limiter.entries[key] = entry
	return true
}

func (rt *_router) wrapAuthRateLimit(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if !rt.authLimiter.allow(host, time.Now()) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "Troppi tentativi, riprova tra un minuto")
			return
		}
		handler(w, r, ps)
	}
}
