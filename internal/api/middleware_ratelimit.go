package api

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

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

func rateLimitMiddleware(rl *ipRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				ip = realIP
			}

			limiter := rl.getLimiter(ip)
			if !limiter.Allow() {
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
