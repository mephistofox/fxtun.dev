package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

var _ store.MagicLinkStore = (*MagicLinkStore)(nil)

// MagicLinkStore implements store.MagicLinkStore backed by Redis.
//
// Layout (all under the client key prefix):
//
//	magiclink:tok:<token>      HASH  {email, code_hash, ip, lang}  TTL=link ttl
//	magiclink:email:<h(email)> STRING token                       TTL=link ttl
//	magiclink:attempt:<h>      INT   failed code attempts          TTL=link ttl
//	magiclink:cd:<h>           STRING "1" (send cooldown marker)   TTL=cooldown
//
// Emails are hashed (not stored in cleartext) in index/counter keys to keep PII
// out of Redis keyspace listings.
type MagicLinkStore struct {
	c *Client
}

// NewMagicLinkStore creates a new Redis-backed magic-link store.
func NewMagicLinkStore(c *Client) *MagicLinkStore {
	return &MagicLinkStore{c: c}
}

func emailKeyHash(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])
}

func (m *MagicLinkStore) tokenKey(token string) string { return m.c.Key("magiclink", "tok", token) }
func (m *MagicLinkStore) emailKey(email string) string {
	return m.c.Key("magiclink", "email", emailKeyHash(email))
}
func (m *MagicLinkStore) attemptKey(email string) string {
	return m.c.Key("magiclink", "attempt", emailKeyHash(email))
}
func (m *MagicLinkStore) cooldownKey(email string) string {
	return m.c.Key("magiclink", "cd", emailKeyHash(email))
}

// Create stores a verification entry and returns its opaque link token.
func (m *MagicLinkStore) Create(entry *store.MagicLinkEntry, ttl time.Duration) (string, error) {
	ctx := context.Background()

	token, err := randomHex(32)
	if err != nil {
		return "", err
	}

	// Best-effort: drop any previous pending token for this email so only the
	// newest link is valid.
	if old, err := m.c.RDB().Get(ctx, m.emailKey(entry.Email)).Result(); err == nil && old != "" {
		_ = m.c.RDB().Del(ctx, m.tokenKey(old)).Err()
	}

	pipe := m.c.RDB().Pipeline()
	pipe.HSet(ctx, m.tokenKey(token), map[string]interface{}{
		"email":     entry.Email,
		"code_hash": entry.CodeHash,
		"ip":        entry.IP,
		"lang":      entry.Lang,
	})
	pipe.Expire(ctx, m.tokenKey(token), ttl)
	pipe.Set(ctx, m.emailKey(entry.Email), token, ttl)
	pipe.Del(ctx, m.attemptKey(entry.Email))
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return token, nil
}

// ConsumeToken atomically retrieves and deletes the entry for a link token.
func (m *MagicLinkStore) ConsumeToken(token string) *store.MagicLinkEntry {
	ctx := context.Background()

	result, err := consumeScript.Run(ctx, m.c.RDB(), []string{m.tokenKey(token)}).StringSlice()
	if err != nil || len(result) == 0 {
		return nil
	}
	vals := sliceToMap(result)
	if len(vals) == 0 {
		return nil
	}

	entry := &store.MagicLinkEntry{
		Email:    vals["email"],
		CodeHash: vals["code_hash"],
		IP:       vals["ip"],
		Lang:     vals["lang"],
	}
	// Clear the secondary index and attempt counter best-effort.
	_ = m.c.RDB().Del(ctx, m.emailKey(entry.Email), m.attemptKey(entry.Email)).Err()
	return entry
}

// LookupByToken returns the entry for a link token without consuming it.
func (m *MagicLinkStore) LookupByToken(token string) (*store.MagicLinkEntry, bool) {
	ctx := context.Background()
	vals, err := m.c.RDB().HGetAll(ctx, m.tokenKey(token)).Result()
	if err != nil || len(vals) == 0 {
		return nil, false
	}
	return &store.MagicLinkEntry{
		Email:    vals["email"],
		CodeHash: vals["code_hash"],
		IP:       vals["ip"],
		Lang:     vals["lang"],
	}, true
}

// LookupByEmail returns the active token and entry for an email without consuming it.
func (m *MagicLinkStore) LookupByEmail(email string) (string, *store.MagicLinkEntry, bool) {
	ctx := context.Background()

	token, err := m.c.RDB().Get(ctx, m.emailKey(email)).Result()
	if err != nil || token == "" {
		return "", nil, false
	}
	vals, err := m.c.RDB().HGetAll(ctx, m.tokenKey(token)).Result()
	if err != nil || len(vals) == 0 {
		return "", nil, false
	}
	return token, &store.MagicLinkEntry{
		Email:    vals["email"],
		CodeHash: vals["code_hash"],
		IP:       vals["ip"],
		Lang:     vals["lang"],
	}, true
}

// IncrAttempt increments and returns the failed-code attempt counter for email.
func (m *MagicLinkStore) IncrAttempt(email string, ttl time.Duration) int {
	ctx := context.Background()
	pipe := m.c.RDB().Pipeline()
	incr := pipe.Incr(ctx, m.attemptKey(email))
	pipe.Expire(ctx, m.attemptKey(email), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0
	}
	return int(incr.Val())
}

// AllowSend reports whether a new link may be sent to email now, recording a
// cooldown when it returns true.
func (m *MagicLinkStore) AllowSend(email string, cooldown time.Duration) bool {
	ctx := context.Background()
	ok, err := m.c.RDB().SetNX(ctx, m.cooldownKey(email), "1", cooldown).Result()
	if err != nil {
		// Fail open on Redis errors rather than locking everyone out; the IP
		// rate limiter still applies.
		return true
	}
	return ok
}
