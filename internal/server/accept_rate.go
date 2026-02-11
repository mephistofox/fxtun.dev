package server

import (
	"sync"
	"sync/atomic"

	"golang.org/x/time/rate"
)

// acceptRateLimiter provides global and per-IP rate limiting for accept().
// Authenticated client IPs are trusted and bypass per-IP checks.
type acceptRateLimiter struct {
	global *rate.Limiter
	perIP  map[string]*rate.Limiter
	mu     sync.Mutex
	limit  rate.Limit
	burst  int

	// trusted holds reference-counted IPs of authenticated clients.
	// These IPs bypass per-IP rate limiting (data sessions need many connections).
	trusted sync.Map // map[string]*int32
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
	// Check global rate (always applies)
	if !a.global.Allow() {
		return false
	}

	// Trusted IPs (authenticated clients) bypass per-IP rate limiting
	if a.isTrusted(ip) {
		return true
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

// Trust marks an IP as trusted (authenticated client connected).
// Call when a client successfully authenticates.
func (a *acceptRateLimiter) Trust(ip string) {
	val, _ := a.trusted.LoadOrStore(ip, new(int32))
	atomic.AddInt32(val.(*int32), 1)
}

// Untrust removes trust for an IP (client disconnected).
// Call when a client disconnects.
func (a *acceptRateLimiter) Untrust(ip string) {
	val, ok := a.trusted.Load(ip)
	if !ok {
		return
	}
	if atomic.AddInt32(val.(*int32), -1) <= 0 {
		a.trusted.Delete(ip)
	}
}

func (a *acceptRateLimiter) isTrusted(ip string) bool {
	val, ok := a.trusted.Load(ip)
	if !ok {
		return false
	}
	return atomic.LoadInt32(val.(*int32)) > 0
}

// Cleanup removes stale per-IP limiters. Call periodically.
func (a *acceptRateLimiter) Cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Simple strategy: remove all entries; they'll be recreated on demand.
	// This prevents unbounded growth of the map.
	a.perIP = make(map[string]*rate.Limiter)
}
