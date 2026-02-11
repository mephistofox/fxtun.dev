package server

import (
	"sync"

	"golang.org/x/time/rate"
)

// acceptRateLimiter provides global and per-IP rate limiting for accept().
type acceptRateLimiter struct {
	global *rate.Limiter
	perIP  map[string]*rate.Limiter
	mu     sync.Mutex
	limit  rate.Limit
	burst  int
}

// newAcceptRateLimiter creates a new accept rate limiter.
// globalRate and perIPRate are in connections per second.
func newAcceptRateLimiter(globalRate, perIPRate int) *acceptRateLimiter {
	if globalRate <= 0 {
		globalRate = 50
	}
	if perIPRate <= 0 {
		perIPRate = 5
	}
	return &acceptRateLimiter{
		global: rate.NewLimiter(rate.Limit(globalRate), globalRate),
		perIP:  make(map[string]*rate.Limiter),
		limit:  rate.Limit(perIPRate),
		burst:  perIPRate,
	}
}

// Allow returns true if the connection from ip is allowed.
func (a *acceptRateLimiter) Allow(ip string) bool {
	// Check global rate
	if !a.global.Allow() {
		return false
	}

	// Check per-IP rate
	a.mu.Lock()
	lim, ok := a.perIP[ip]
	if !ok {
		lim = rate.NewLimiter(a.limit, a.burst)
		a.perIP[ip] = lim
	}
	a.mu.Unlock()

	return lim.Allow()
}

// Cleanup removes stale per-IP limiters. Call periodically.
func (a *acceptRateLimiter) Cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Simple strategy: remove all entries; they'll be recreated on demand.
	// This prevents unbounded growth of the map.
	a.perIP = make(map[string]*rate.Limiter)
}
