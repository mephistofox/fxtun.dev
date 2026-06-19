package redis

import (
	"context"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

var _ store.TLSCache = (*TLSCache)(nil)

// TLSCache implements store.TLSCache backed by Redis.
type TLSCache struct {
	c *Client
}

// NewTLSCache creates a new Redis-backed TLS certificate cache.
func NewTLSCache(c *Client) *TLSCache {
	return &TLSCache{c: c}
}

// Get retrieves a cached TLS certificate for the given domain.
func (t *TLSCache) Get(domain string) (certPEM, keyPEM []byte, err error) {
	ctx := context.Background()
	key := t.c.Key("tls", domain)

	vals, err := t.c.RDB().HGetAll(ctx, key).Result()
	if err != nil {
		return nil, nil, err
	}
	if len(vals) == 0 {
		return nil, nil, nil
	}

	return []byte(vals["cert_pem"]), []byte(vals["key_pem"]), nil
}

// Put stores a TLS certificate with TTL based on its expiration time.
func (t *TLSCache) Put(domain string, certPEM, keyPEM []byte, expiresAt time.Time) error {
	ctx := context.Background()
	key := t.c.Key("tls", domain)

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // already expired, don't store
	}

	fields := map[string]interface{}{
		"cert_pem": string(certPEM),
		"key_pem":  string(keyPEM),
	}

	pipe := t.c.RDB().Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.ExpireAt(ctx, key, expiresAt)

	_, err := pipe.Exec(ctx)
	return err
}

// Delete removes a cached TLS certificate.
func (t *TLSCache) Delete(domain string) error {
	ctx := context.Background()
	return t.c.RDB().Del(ctx, t.c.Key("tls", domain)).Err()
}
