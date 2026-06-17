package redis

import (
	"context"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

var _ store.TunnelRegistry = (*TunnelRegistry)(nil)

const tunnelTTL = 60 * time.Second

// TunnelRegistry implements store.TunnelRegistry backed by Redis.
type TunnelRegistry struct {
	c        *Client
	serverID string
}

// NewTunnelRegistry creates a new Redis-backed tunnel registry.
func NewTunnelRegistry(c *Client, serverID string) *TunnelRegistry {
	return &TunnelRegistry{c: c, serverID: serverID}
}

// Register stores a tunnel entry in Redis with TTL-based expiration.
func (t *TunnelRegistry) Register(entry store.TunnelEntry) error {
	ctx := context.Background()
	rdb := t.c.RDB()

	infoKey := t.c.Key("tunnel", "info", entry.TunnelID)
	userSetKey := t.c.Key("tunnel", "user", strconv.FormatInt(entry.UserID, 10))

	entry.ServerID = t.serverID
	fields := tunnelToMap(entry)

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, infoKey, fields)
	pipe.Expire(ctx, infoKey, tunnelTTL)
	pipe.SAdd(ctx, userSetKey, entry.TunnelID)

	if entry.Subdomain != "" {
		subKey := t.c.Key("tunnel", "sub", entry.Subdomain)
		pipe.HSet(ctx, subKey, fields)
		pipe.Expire(ctx, subKey, tunnelTTL)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Unregister removes a tunnel entry from Redis.
func (t *TunnelRegistry) Unregister(tunnelID string) error {
	ctx := context.Background()
	rdb := t.c.RDB()
	infoKey := t.c.Key("tunnel", "info", tunnelID)

	// Read entry to get subdomain and user_id for cleanup
	vals, err := rdb.HGetAll(ctx, infoKey).Result()
	if err != nil || len(vals) == 0 {
		return nil
	}

	pipe := rdb.Pipeline()
	pipe.Del(ctx, infoKey)

	if sub := vals["subdomain"]; sub != "" {
		pipe.Del(ctx, t.c.Key("tunnel", "sub", sub))
	}

	if uid := vals["user_id"]; uid != "" {
		pipe.SRem(ctx, t.c.Key("tunnel", "user", uid), tunnelID)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// LookupBySubdomain finds a tunnel entry by its subdomain.
func (t *TunnelRegistry) LookupBySubdomain(subdomain string) (*store.TunnelEntry, error) {
	ctx := context.Background()
	key := t.c.Key("tunnel", "sub", subdomain)

	vals, err := t.c.RDB().HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, nil
	}

	entry, err := mapToTunnel(vals)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// ListByUser returns all tunnel entries for a user.
func (t *TunnelRegistry) ListByUser(userID int64) ([]store.TunnelEntry, error) {
	ctx := context.Background()
	rdb := t.c.RDB()
	userSetKey := t.c.Key("tunnel", "user", strconv.FormatInt(userID, 10))

	tunnelIDs, err := rdb.SMembers(ctx, userSetKey).Result()
	if err != nil {
		return nil, err
	}
	if len(tunnelIDs) == 0 {
		return nil, nil
	}

	pipe := rdb.Pipeline()
	cmds := make([]*goredis.MapStringStringCmd, len(tunnelIDs))
	for i, id := range tunnelIDs {
		cmds[i] = pipe.HGetAll(ctx, t.c.Key("tunnel", "info", id))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		return nil, err
	}

	var entries []store.TunnelEntry
	var stale []string

	for i, cmd := range cmds {
		vals, _ := cmd.Result()
		if len(vals) == 0 {
			stale = append(stale, tunnelIDs[i])
			continue
		}
		entry, err := mapToTunnel(vals)
		if err != nil {
			stale = append(stale, tunnelIDs[i])
			continue
		}
		entries = append(entries, *entry)
	}

	// Clean up stale references
	if len(stale) > 0 {
		sremArgs := make([]interface{}, len(stale))
		for i, v := range stale {
			sremArgs[i] = v
		}
		rdb.SRem(ctx, userSetKey, sremArgs...)
	}

	return entries, nil
}

// Heartbeat refreshes the TTL on a tunnel entry.
func (t *TunnelRegistry) Heartbeat(tunnelID string) error {
	ctx := context.Background()
	rdb := t.c.RDB()
	infoKey := t.c.Key("tunnel", "info", tunnelID)

	// Read subdomain to also refresh its key
	subdomain, err := rdb.HGet(ctx, infoKey, "subdomain").Result()
	if err != nil && err != goredis.Nil {
		return err
	}

	pipe := rdb.Pipeline()
	pipe.Expire(ctx, infoKey, tunnelTTL)

	if subdomain != "" {
		pipe.Expire(ctx, t.c.Key("tunnel", "sub", subdomain), tunnelTTL)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func tunnelToMap(e store.TunnelEntry) map[string]interface{} {
	return map[string]interface{}{
		"tunnel_id":   e.TunnelID,
		"type":        e.Type,
		"name":        e.Name,
		"subdomain":   e.Subdomain,
		"remote_port": strconv.Itoa(e.RemotePort),
		"local_port":  strconv.Itoa(e.LocalPort),
		"client_id":   e.ClientID,
		"user_id":     strconv.FormatInt(e.UserID, 10),
		"server_id":   e.ServerID,
		"created_at":  e.CreatedAt.Format(time.RFC3339),
	}
}

func mapToTunnel(vals map[string]string) (*store.TunnelEntry, error) {
	remotePort, _ := strconv.Atoi(vals["remote_port"])
	localPort, _ := strconv.Atoi(vals["local_port"])
	userID, _ := strconv.ParseInt(vals["user_id"], 10, 64)
	createdAt, _ := time.Parse(time.RFC3339, vals["created_at"])

	return &store.TunnelEntry{
		TunnelID:   vals["tunnel_id"],
		Type:       vals["type"],
		Name:       vals["name"],
		Subdomain:  vals["subdomain"],
		RemotePort: remotePort,
		LocalPort:  localPort,
		ClientID:   vals["client_id"],
		UserID:     userID,
		ServerID:   vals["server_id"],
		CreatedAt:  createdAt,
	}, nil
}
