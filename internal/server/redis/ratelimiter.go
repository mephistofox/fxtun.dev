package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
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

// incrWithTTLScript increments the window counter and sets its TTL in one step.
// Doing this in two round-trips can leave a counter without a TTL, which bans
// the IP forever and leaks the key.
var incrWithTTLScript = goredis.NewScript(`
local c = redis.call('INCR', KEYS[1])
if c == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return c
`)

// Allow returns true if the request from the given IP should be permitted.
func (r *RateLimiter) Allow(ip string) bool {
	ctx := context.Background()
	key := r.c.Key("rl", r.scope, ip)

	count, err := incrWithTTLScript.Run(ctx, r.c.RDB(), []string{key}, int64(rateLimitWindow.Seconds())).Int64()
	if err != nil {
		return false // fail closed on Redis error
	}

	return count <= int64(r.perMinute)
}
