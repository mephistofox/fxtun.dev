package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/auth"
	"github.com/mephistofox/fxtun.dev/internal/server/store"
	"golang.org/x/time/rate"
)

// Compile-time check that ipRateLimiter implements store.RateChecker.
var _ store.RateChecker = (*ipRateLimiter)(nil)

// loginAttemptsPerMin caps login attempts per source IP, slowing password /
// TOTP brute-force beyond the broader auth-group rate limit.
const loginAttemptsPerMin = 8

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	limiters sync.Map
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

func newIPRateLimiter(perMinute int) *ipRateLimiter {
	return &ipRateLimiter{
		rate:  rate.Limit(float64(perMinute) / 60.0),
		burst: perMinute,
		ttl:   10 * time.Minute,
	}
}

func (rl *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	now := time.Now()
	if v, ok := rl.limiters.Load(ip); ok {
		entry := v.(*limiterEntry)
		entry.lastSeen = now
		return entry.limiter
	}
	entry := &limiterEntry{
		limiter:  rate.NewLimiter(rl.rate, rl.burst),
		lastSeen: now,
	}
	if actual, loaded := rl.limiters.LoadOrStore(ip, entry); loaded {
		entry = actual.(*limiterEntry)
		entry.lastSeen = now
		return entry.limiter
	}
	return entry.limiter
}

// Allow implements store.RateChecker.
func (rl *ipRateLimiter) Allow(ip string) bool {
	return rl.getLimiter(ip).Allow()
}

// cleanup removes stale limiters periodically based on TTL
func (rl *ipRateLimiter) cleanup(stopCh <-chan struct{}, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				now := time.Now()
				rl.limiters.Range(func(key, value any) bool {
					entry := value.(*limiterEntry)
					if now.Sub(entry.lastSeen) > rl.ttl {
						rl.limiters.Delete(key)
					}
					return true
				})
			}
		}
	}()
}

func rateLimitMiddleware(rl store.RateChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Key on the real client IP. trustedRealIPMiddleware has already
			// rewritten r.RemoteAddr to the client IP when the request came
			// through a trusted proxy, and ignores forwarded headers from
			// untrusted sources — so this is spoof-safe. Keying on the raw TCP
			// source instead would collapse every client behind nginx's reused
			// keepalive upstream connections into a single shared bucket.
			ip := auth.GetClientIP(r)

			if !rl.Allow(ip) {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
