package redis

import (
	"context"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

var _ store.RateChecker = (*RateLimiter)(nil)

const rateLimitWindow = 60 * time.Second

// RateLimiter implements store.RateChecker using a fixed-window counter in Redis.
type RateLimiter struct {
	c        *Client
	scope    string
	perMinute int
}

// NewRateLimiter creates a new Redis-backed rate limiter.
func NewRateLimiter(c *Client, scope string, perMinute int) *RateLimiter {
	return &RateLimiter{c: c, scope: scope, perMinute: perMinute}
}

// Allow returns true if the request from the given IP should be permitted.
func (r *RateLimiter) Allow(ip string) bool {
	ctx := context.Background()
	key := r.c.Key("rl", r.scope, ip)
	rdb := r.c.RDB()

	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		// On Redis error, allow the request (fail open)
		return true
	}

	if count == 1 {
		rdb.Expire(ctx, key, rateLimitWindow)
	}

	return count <= int64(r.perMinute)
}
