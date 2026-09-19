package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

var _ store.DeviceStore = (*DeviceStore)(nil)

const deviceSessionTTL = 5 * time.Minute

// DeviceStore implements store.DeviceStore backed by Redis.
type DeviceStore struct {
	c *Client
}

// NewDeviceStore creates a new Redis-backed device store.
func NewDeviceStore(c *Client) *DeviceStore {
	return &DeviceStore{c: c}
}

// Create generates a new device session with a random ID.
func (d *DeviceStore) Create() (*store.DeviceSession, error) {
	ctx := context.Background()

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(b)

	userCode, err := generateUserCode()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	key := d.c.Key("device", id)

	fields := map[string]interface{}{
		"status":     "pending",
		"token":      "",
		"user_code":  userCode,
		"created_at": now.Format(time.RFC3339),
	}

	pipe := d.c.RDB().Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, deviceSessionTTL)
	// Reverse index so the approval page can find the session from the code
	// the user typed, without ever learning the polling secret.
	pipe.Set(ctx, d.c.Key("devicecode", userCode), id, deviceSessionTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	return &store.DeviceSession{
		ID:        id,
		UserCode:  userCode,
		Status:    "pending",
		Token:     "",
		CreatedAt: now,
	}, nil
}

// userCodeAlphabet omits characters that are easy to confuse when read off a
// terminal and typed into a browser (0/O, 1/I/L, U/V).
const userCodeAlphabet = "BCDFGHJKMNPQRSTWXYZ23456789"

// generateUserCode returns a code in the form XXXX-XXXX.
func generateUserCode() (string, error) {
	const n = 8
	out := make([]byte, 0, n+1)
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, v := range buf {
		if i == n/2 {
			out = append(out, '-')
		}
		out = append(out, userCodeAlphabet[int(v)%len(userCodeAlphabet)])
	}
	return string(out), nil
}

// GetByUserCode resolves a user-typed code to its session.
func (d *DeviceStore) GetByUserCode(userCode string) *store.DeviceSession {
	ctx := context.Background()
	id, err := d.c.RDB().Get(ctx, d.c.Key("devicecode", userCode)).Result()
	if err != nil || id == "" {
		return nil
	}
	return d.Get(id)
}

// Get retrieves a device session by ID. Returns nil if not found.
func (d *DeviceStore) Get(id string) *store.DeviceSession {
	ctx := context.Background()
	key := d.c.Key("device", id)

	vals, err := d.c.RDB().HGetAll(ctx, key).Result()
	if err != nil || len(vals) == 0 {
		return nil
	}

	createdAt, err := time.Parse(time.RFC3339, vals["created_at"])
	if err != nil {
		return nil
	}

	return &store.DeviceSession{
		ID:        id,
		UserCode:  vals["user_code"],
		Status:    vals["status"],
		Token:     vals["token"],
		CreatedAt: createdAt,
	}
}

// authorizeScript atomically checks key existence and sets status+token.
var authorizeScript = goredis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return 0 end
redis.call('HSET', KEYS[1], 'status', ARGV[1], 'token', ARGV[2])
return 1
`)

// Authorize marks a device session as authorized with the given token.
// Returns false if the session no longer exists.
func (d *DeviceStore) Authorize(id, token string) bool {
	ctx := context.Background()
	key := d.c.Key("device", id)

	result, err := authorizeScript.Run(ctx, d.c.RDB(), []string{key}, "authorized", token).Int()
	if err != nil {
		return false
	}
	return result == 1
}

// Delete removes a device session and its user-code index.
func (d *DeviceStore) Delete(id string) {
	ctx := context.Background()
	if sess := d.Get(id); sess != nil && sess.UserCode != "" {
		d.c.RDB().Del(ctx, d.c.Key("devicecode", sess.UserCode))
	}
	d.c.RDB().Del(ctx, d.c.Key("device", id))
}
