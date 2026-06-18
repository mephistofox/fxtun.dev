package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mephistofox/fxtunnel/internal/server/database"
	"github.com/mephistofox/fxtunnel/internal/server/store"
)

var (
	_ store.SessionStore        = (*SessionStore)(nil)
	_ store.RotatedTokenTracker = (*SessionStore)(nil)
)

const maxSessionTTL = 7 * 24 * time.Hour // 7 days

// SessionStore implements store.SessionStore backed by Redis.
type SessionStore struct {
	c *Client
}

// NewSessionStore creates a new Redis-backed session store.
func NewSessionStore(c *Client) *SessionStore {
	return &SessionStore{c: c}
}

// Create stores a session in Redis with TTL based on ExpiresAt.
func (s *SessionStore) Create(session *database.Session) error {
	ctx := context.Background()
	rdb := s.c.RDB()

	key := s.c.Key("session", session.RefreshTokenHash)
	userSetKey := s.c.Key("user", "sessions", strconv.FormatInt(session.UserID, 10))

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return nil // already expired, don't store
	}

	fields := map[string]interface{}{
		"user_id":    strconv.FormatInt(session.UserID, 10),
		"user_agent": session.UserAgent,
		"ip_address": session.IPAddress,
		"created_at": session.CreatedAt.Format(time.RFC3339),
		"expires_at": session.ExpiresAt.Format(time.RFC3339),
	}

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.ExpireAt(ctx, key, session.ExpiresAt)
	pipe.SAdd(ctx, userSetKey, session.RefreshTokenHash)
	pipe.Expire(ctx, userSetKey, maxSessionTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// GetByTokenHash retrieves a session by its refresh token hash.
func (s *SessionStore) GetByTokenHash(tokenHash string) (*database.Session, error) {
	ctx := context.Background()
	key := s.c.Key("session", tokenHash)

	vals, err := s.c.RDB().HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, database.ErrSessionNotFound
	}

	return parseSession(tokenHash, vals)
}

// GetByUserID retrieves all sessions for a user.
func (s *SessionStore) GetByUserID(userID int64) ([]*database.Session, error) {
	ctx := context.Background()
	rdb := s.c.RDB()
	userSetKey := s.c.Key("user", "sessions", strconv.FormatInt(userID, 10))

	hashes, err := rdb.SMembers(ctx, userSetKey).Result()
	if err != nil {
		return nil, err
	}
	if len(hashes) == 0 {
		return nil, nil
	}

	// Pipeline HGETALL for each token hash
	pipe := rdb.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(hashes))
	for i, h := range hashes {
		cmds[i] = pipe.HGetAll(ctx, s.c.Key("session", h))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	var sessions []*database.Session
	var stale []string

	for i, cmd := range cmds {
		vals, _ := cmd.Result()
		if len(vals) == 0 {
			stale = append(stale, hashes[i])
			continue
		}
		sess, err := parseSession(hashes[i], vals)
		if err != nil {
			stale = append(stale, hashes[i])
			continue
		}
		sessions = append(sessions, sess)
	}

	// Clean up stale references
	if len(stale) > 0 {
		sremArgs := make([]interface{}, len(stale))
		for i, v := range stale {
			sremArgs[i] = v
		}
		rdb.SRem(ctx, userSetKey, sremArgs...)
	}

	return sessions, nil
}

// Delete is a no-op for Redis. Callers should use DeleteByTokenHash instead.
func (s *SessionStore) Delete(_ int64) error {
	return nil
}

// DeleteByTokenHash removes a session by its refresh token hash.
func (s *SessionStore) DeleteByTokenHash(tokenHash string) error {
	ctx := context.Background()
	rdb := s.c.RDB()
	key := s.c.Key("session", tokenHash)

	// Read user_id so we can clean the user set
	userIDStr, err := rdb.HGet(ctx, key, "user_id").Result()
	if err != nil && err != redis.Nil {
		return err
	}

	pipe := rdb.Pipeline()
	pipe.Del(ctx, key)
	if userIDStr != "" {
		userSetKey := s.c.Key("user", "sessions", userIDStr)
		pipe.SRem(ctx, userSetKey, tokenHash)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// DeleteByUserID removes all sessions for a user.
func (s *SessionStore) DeleteByUserID(userID int64) error {
	ctx := context.Background()
	rdb := s.c.RDB()
	userSetKey := s.c.Key("user", "sessions", strconv.FormatInt(userID, 10))

	hashes, err := rdb.SMembers(ctx, userSetKey).Result()
	if err != nil {
		return err
	}

	if len(hashes) > 0 {
		keys := make([]string, len(hashes))
		for i, h := range hashes {
			keys[i] = s.c.Key("session", h)
		}
		pipe := rdb.Pipeline()
		pipe.Del(ctx, keys...)
		pipe.Del(ctx, userSetKey)
		_, err = pipe.Exec(ctx)
		return err
	}

	return rdb.Del(ctx, userSetKey).Err()
}

// DeleteExpired is a no-op — Redis TTL handles expiration automatically.
func (s *SessionStore) DeleteExpired() (int64, error) {
	return 0, nil
}

// MarkRotated records a rotated refresh-token hash with the owning user, kept
// for ttl so a later reuse of the same token can be detected.
func (s *SessionStore) MarkRotated(tokenHash string, userID int64, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	ctx := context.Background()
	key := s.c.Key("session", "rotated", tokenHash)
	return s.c.RDB().Set(ctx, key, strconv.FormatInt(userID, 10), ttl).Err()
}

// RotatedOwner returns the user a recently rotated token belonged to.
func (s *SessionStore) RotatedOwner(tokenHash string) (int64, bool, error) {
	ctx := context.Background()
	key := s.c.Key("session", "rotated", tokenHash)
	v, err := s.c.RDB().Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	userID, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return userID, true, nil
}

// parseSession converts a Redis hash map into a database.Session.
func parseSession(tokenHash string, vals map[string]string) (*database.Session, error) {
	userID, err := strconv.ParseInt(vals["user_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339, vals["created_at"])
	if err != nil {
		return nil, err
	}
	expiresAt, err := time.Parse(time.RFC3339, vals["expires_at"])
	if err != nil {
		return nil, err
	}

	return &database.Session{
		UserID:           userID,
		RefreshTokenHash: tokenHash,
		UserAgent:        vals["user_agent"],
		IPAddress:        vals["ip_address"],
		CreatedAt:        createdAt,
		ExpiresAt:        expiresAt,
	}, nil
}
