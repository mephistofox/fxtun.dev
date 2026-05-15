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

	now := time.Now().UTC()
	key := d.c.Key("device", id)

	fields := map[string]interface{}{
		"status":     "pending",
		"token":      "",
		"created_at": now.Format(time.RFC3339),
	}

	pipe := d.c.RDB().Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, deviceSessionTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	return &store.DeviceSession{
		ID:        id,
		Status:    "pending",
		Token:     "",
		CreatedAt: now,
	}, nil
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

// Delete removes a device session.
func (d *DeviceStore) Delete(id string) {
	ctx := context.Background()
	d.c.RDB().Del(ctx, d.c.Key("device", id))
}
