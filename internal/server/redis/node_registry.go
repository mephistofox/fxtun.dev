package redis

import (
	"context"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

var _ store.NodeRegistry = (*NodeRegistry)(nil)

const nodeTTL = 90 * time.Second

// NodeRegistry implements store.NodeRegistry backed by Redis.
type NodeRegistry struct {
	c *Client
}

// NewNodeRegistry creates a new Redis-backed node registry.
func NewNodeRegistry(c *Client) *NodeRegistry {
	return &NodeRegistry{c: c}
}

// RegisterNode stores a node entry in Redis.
func (r *NodeRegistry) RegisterNode(entry store.NodeEntry) error {
	ctx := context.Background()
	rdb := r.c.RDB()

	infoKey := r.c.Key("node", "info", entry.NodeID)
	activeSetKey := r.c.Key("node", "active")

	fields := nodeToMap(entry)

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, infoKey, fields)
	pipe.Expire(ctx, infoKey, nodeTTL)
	pipe.SAdd(ctx, activeSetKey, entry.NodeID)

	_, err := pipe.Exec(ctx)
	return err
}

// UnregisterNode removes a node from Redis.
func (r *NodeRegistry) UnregisterNode(nodeID string) error {
	ctx := context.Background()
	rdb := r.c.RDB()

	pipe := rdb.Pipeline()
	pipe.Del(ctx, r.c.Key("node", "info", nodeID))
	pipe.SRem(ctx, r.c.Key("node", "active"), nodeID)

	_, err := pipe.Exec(ctx)
	return err
}

// GetNode retrieves a single node by ID.
func (r *NodeRegistry) GetNode(nodeID string) (*store.NodeEntry, error) {
	ctx := context.Background()
	key := r.c.Key("node", "info", nodeID)

	vals, err := r.c.RDB().HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, nil
	}

	return mapToNode(vals), nil
}

// ListActiveNodes returns all active nodes from Redis.
func (r *NodeRegistry) ListActiveNodes() ([]store.NodeEntry, error) {
	ctx := context.Background()
	rdb := r.c.RDB()
	activeSetKey := r.c.Key("node", "active")

	nodeIDs, err := rdb.SMembers(ctx, activeSetKey).Result()
	if err != nil {
		return nil, err
	}
	if len(nodeIDs) == 0 {
		return nil, nil
	}

	pipe := rdb.Pipeline()
	cmds := make([]*goredis.MapStringStringCmd, len(nodeIDs))
	for i, id := range nodeIDs {
		cmds[i] = pipe.HGetAll(ctx, r.c.Key("node", "info", id))
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		return nil, err
	}

	var entries []store.NodeEntry
	var stale []string

	for i, cmd := range cmds {
		vals, _ := cmd.Result()
		if len(vals) == 0 {
			stale = append(stale, nodeIDs[i])
			continue
		}
		entries = append(entries, *mapToNode(vals))
	}

	// Clean up stale references
	if len(stale) > 0 {
		sremArgs := make([]interface{}, len(stale))
		for i, v := range stale {
			sremArgs[i] = v
		}
		rdb.SRem(ctx, activeSetKey, sremArgs...)
	}

	return entries, nil
}

// HeartbeatNode refreshes the TTL and updates stats for a node.
func (r *NodeRegistry) HeartbeatNode(nodeID string, tunnelCount, clientCount int) error {
	ctx := context.Background()
	rdb := r.c.RDB()
	infoKey := r.c.Key("node", "info", nodeID)

	activeSetKey := r.c.Key("node", "active")

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, infoKey,
		"tunnel_count", strconv.Itoa(tunnelCount),
		"client_count", strconv.Itoa(clientCount),
		"updated_at", time.Now().Format(time.RFC3339),
	)
	pipe.Expire(ctx, infoKey, nodeTTL)
	pipe.SAdd(ctx, activeSetKey, nodeID) // ensure node stays in active set

	_, err := pipe.Exec(ctx)
	return err
}

func nodeToMap(e store.NodeEntry) map[string]interface{} {
	return map[string]interface{}{
		"node_id":      e.NodeID,
		"name":         e.Name,
		"region":       e.Region,
		"public_addr":  e.PublicAddr,
		"http_addr":    e.HTTPAddr,
		"status":       e.Status,
		"tunnel_count": strconv.Itoa(e.TunnelCount),
		"client_count": strconv.Itoa(e.ClientCount),
		"updated_at":   e.UpdatedAt.Format(time.RFC3339),
	}
}

func mapToNode(vals map[string]string) *store.NodeEntry {
	tunnelCount, _ := strconv.Atoi(vals["tunnel_count"])
	clientCount, _ := strconv.Atoi(vals["client_count"])
	updatedAt, _ := time.Parse(time.RFC3339, vals["updated_at"])

	return &store.NodeEntry{
		NodeID:      vals["node_id"],
		Name:        vals["name"],
		Region:      vals["region"],
		PublicAddr:  vals["public_addr"],
		HTTPAddr:    vals["http_addr"],
		Status:      vals["status"],
		TunnelCount: tunnelCount,
		ClientCount: clientCount,
		UpdatedAt:   updatedAt,
	}
}
