package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

var _ store.IPBanStore = (*IPBanStore)(nil)

// IPBanStore stores temporary IP bans in Redis with key TTL.
//
// Key layout: <prefix>ipban:<ip>  →  JSON{reason, banned_at_unix}
// TTL on the key implements automatic expiry; the JSON payload holds the
// human-readable reason and the original ban timestamp.
type IPBanStore struct {
	c *Client
}

// NewIPBanStore creates a new Redis-backed IP ban store.
func NewIPBanStore(c *Client) *IPBanStore {
	return &IPBanStore{c: c}
}

type ipBanPayload struct {
	Reason       string `json:"reason"`
	BannedAtUnix int64  `json:"banned_at_unix"`
}

func (s *IPBanStore) key(ip string) string {
	return s.c.Key("ipban", ip)
}

// Ban records the IP as banned. Returns true if this is a new ban.
//
// Uses SET NX to atomically claim a new ban (avoids the TOCTOU race where two
// concurrent callers both think they created the ban and both fire a Telegram
// alert). For an already-banned IP we only refresh the TTL — the original
// banned_at/reason payload is preserved.
func (s *IPBanStore) Ban(ip, reason string, ttl time.Duration) (bool, error) {
	if ip == "" || ttl <= 0 {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rdb := s.c.RDB()
	key := s.key(ip)

	data, err := json.Marshal(ipBanPayload{Reason: reason, BannedAtUnix: time.Now().UTC().Unix()})
	if err != nil {
		return false, err
	}

	created, err := rdb.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return false, err
	}
	if created {
		return true, nil
	}
	// Already banned — refresh the expiry, keep the original payload.
	if err := rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return false, err
	}
	return false, nil
}

// IsBanned returns whether the IP is currently banned and the ban reason.
func (s *IPBanStore) IsBanned(ip string) (bool, string, error) {
	if ip == "" {
		return false, "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := s.c.RDB().Get(ctx, s.key(ip)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return false, "", nil
		}
		return false, "", err
	}
	var payload ipBanPayload
	if jerr := json.Unmarshal(raw, &payload); jerr != nil {
		// Corrupt payload but key exists → fail closed: treat as banned.
		return true, "", nil
	}
	return true, payload.Reason, nil
}

// Unban removes the ban for the given IP.
func (s *IPBanStore) Unban(ip string) error {
	if ip == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.c.RDB().Del(ctx, s.key(ip)).Err()
}

// List returns all currently active bans by scanning the keyspace.
func (s *IPBanStore) List() ([]store.IPBanEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rdb := s.c.RDB()
	pattern := s.c.Key("ipban", "*")

	var entries []store.IPBanEntry
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return entries, err
		}
		for _, key := range keys {
			raw, err := rdb.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			ttl, err := rdb.TTL(ctx, key).Result()
			if err != nil || ttl <= 0 {
				continue
			}
			var payload ipBanPayload
			_ = json.Unmarshal(raw, &payload)

			ip := key
			if prefix := s.c.Key("ipban", ""); len(prefix) > 0 && len(key) > len(prefix) {
				ip = key[len(prefix):]
			}

			bannedAt := time.Unix(payload.BannedAtUnix, 0).UTC()
			entries = append(entries, store.IPBanEntry{
				IP:        ip,
				Reason:    payload.Reason,
				BannedAt:  bannedAt,
				ExpiresAt: time.Now().UTC().Add(ttl),
			})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return entries, nil
}
