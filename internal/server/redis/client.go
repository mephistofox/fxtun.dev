package redis

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/config"
)

// Client wraps a go-redis client with key prefixing and logging.
type Client struct {
	rdb    redis.UniversalClient
	prefix string
	log    zerolog.Logger
}

// New creates a Redis client based on config (standalone or Sentinel).
func New(cfg config.RedisSettings, log zerolog.Logger) (*Client, error) {
	l := log.With().Str("component", "redis").Logger()

	var rdb redis.UniversalClient
	if cfg.SentinelEnabled {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.SentinelMaster,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
			DialTimeout:   5 * time.Second,
			ReadTimeout:   3 * time.Second,
			WriteTimeout:  3 * time.Second,
		})
		l.Info().Str("master", cfg.SentinelMaster).Strs("sentinels", cfg.SentinelAddrs).Msg("using Sentinel mode")
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		})
		l.Info().Str("addr", cfg.Addr).Msg("using standalone mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, err
	}

	l.Info().Msg("connected")
	return &Client{rdb: rdb, prefix: cfg.KeyPrefix, log: l}, nil
}

// Key builds a prefixed Redis key: prefix + parts joined by ":".
func (c *Client) Key(parts ...string) string {
	return c.prefix + strings.Join(parts, ":")
}

// RDB returns the underlying Redis client for direct operations.
func (c *Client) RDB() redis.UniversalClient {
	return c.rdb
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	c.log.Info().Msg("closing connection")
	return c.rdb.Close()
}
