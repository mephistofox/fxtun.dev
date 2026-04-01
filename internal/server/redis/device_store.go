package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

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

// Authorize marks a device session as authorized with the given token.
// Returns false if the session no longer exists.
func (d *DeviceStore) Authorize(id, token string) bool {
	ctx := context.Background()
	key := d.c.Key("device", id)

	exists, err := d.c.RDB().Exists(ctx, key).Result()
	if err != nil || exists == 0 {
		return false
	}

	err = d.c.RDB().HSet(ctx, key, map[string]interface{}{
		"status": "authorized",
		"token":  token,
	}).Err()

	return err == nil
}

// Delete removes a device session.
func (d *DeviceStore) Delete(id string) {
	ctx := context.Background()
	d.c.RDB().Del(ctx, d.c.Key("device", id))
}
