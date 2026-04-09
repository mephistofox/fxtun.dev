package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EdgeNodeRepository handles edge node database operations.
type EdgeNodeRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new edge node and returns it with generated fields.
func (r *EdgeNodeRepository) Create(node *EdgeNode) error {
	ctx := context.Background()
	err := r.pool.QueryRow(ctx,
		`INSERT INTO edge_nodes (node_id, name, region, public_addr, http_addr, version, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, status, created_at, updated_at`,
		node.NodeID, node.Name, node.Region, node.PublicAddr, node.HTTPAddr, node.Version, node.Metadata,
	).Scan(&node.ID, &node.Status, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create edge node: %w", err)
	}
	return nil
}

// GetByNodeID retrieves a node by its unique node_id.
func (r *EdgeNodeRepository) GetByNodeID(nodeID string) (*EdgeNode, error) {
	ctx := context.Background()
	n := &EdgeNode{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, node_id, name, region, public_addr, http_addr, status,
		        approved_at, approved_by, last_heartbeat_at, version, metadata, created_at, updated_at
		 FROM edge_nodes WHERE node_id = $1`, nodeID,
	).Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.HTTPAddr, &n.Status,
		&n.ApprovedAt, &n.ApprovedBy, &n.LastHeartbeatAt, &n.Version, &n.Metadata, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrEdgeNodeNotFound
		}
		return nil, fmt.Errorf("get edge node by node_id: %w", err)
	}
	return n, nil
}

// GetByID retrieves a node by its database ID.
func (r *EdgeNodeRepository) GetByID(id int64) (*EdgeNode, error) {
	ctx := context.Background()
	n := &EdgeNode{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, node_id, name, region, public_addr, http_addr, status,
		        approved_at, approved_by, last_heartbeat_at, version, metadata, created_at, updated_at
		 FROM edge_nodes WHERE id = $1`, id,
	).Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.HTTPAddr, &n.Status,
		&n.ApprovedAt, &n.ApprovedBy, &n.LastHeartbeatAt, &n.Version, &n.Metadata, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrEdgeNodeNotFound
		}
		return nil, fmt.Errorf("get edge node by id: %w", err)
	}
	return n, nil
}

// List returns nodes filtered by status. Empty status returns all.
func (r *EdgeNodeRepository) List(status string) ([]*EdgeNode, error) {
	ctx := context.Background()
	var rows pgx.Rows
	var err error

	if status != "" {
		rows, err = r.pool.Query(ctx,
			`SELECT id, node_id, name, region, public_addr, http_addr, status,
			        approved_at, approved_by, last_heartbeat_at, version, metadata, created_at, updated_at
			 FROM edge_nodes WHERE status = $1 ORDER BY created_at DESC`, status)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, node_id, name, region, public_addr, http_addr, status,
			        approved_at, approved_by, last_heartbeat_at, version, metadata, created_at, updated_at
			 FROM edge_nodes ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, fmt.Errorf("list edge nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*EdgeNode
	for rows.Next() {
		n := &EdgeNode{}
		if err := rows.Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.HTTPAddr, &n.Status,
			&n.ApprovedAt, &n.ApprovedBy, &n.LastHeartbeatAt, &n.Version, &n.Metadata, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan edge node: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// UpdateStatus changes the node status and records who approved it.
func (r *EdgeNodeRepository) UpdateStatus(id int64, status string, approvedBy int64) error {
	ctx := context.Background()
	now := time.Now()
	tag, err := r.pool.Exec(ctx,
		`UPDATE edge_nodes SET status = $1, approved_at = $2, approved_by = $3, updated_at = $4 WHERE id = $5`,
		status, &now, &approvedBy, &now, id)
	if err != nil {
		return fmt.Errorf("update edge node status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEdgeNodeNotFound
	}
	return nil
}

// UpdateHeartbeat updates the node's last heartbeat time and metadata.
func (r *EdgeNodeRepository) UpdateHeartbeat(nodeID string, metadata string) error {
	ctx := context.Background()
	now := time.Now()
	tag, err := r.pool.Exec(ctx,
		`UPDATE edge_nodes SET last_heartbeat_at = $1, metadata = $2, updated_at = $3 WHERE node_id = $4 AND status = 'active'`,
		&now, metadata, &now, nodeID)
	if err != nil {
		return fmt.Errorf("update edge node heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEdgeNodeNotFound
	}
	return nil
}

// Delete removes a node by ID.
func (r *EdgeNodeRepository) Delete(id int64) error {
	ctx := context.Background()
	tag, err := r.pool.Exec(ctx, `DELETE FROM edge_nodes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete edge node: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEdgeNodeNotFound
	}
	return nil
}

// ListStaleNodes returns active nodes whose heartbeat is older than the given threshold.
func (r *EdgeNodeRepository) ListStaleNodes(threshold time.Duration) ([]*EdgeNode, error) {
	ctx := context.Background()
	cutoff := time.Now().Add(-threshold)
	rows, err := r.pool.Query(ctx,
		`SELECT id, node_id, name, region, public_addr, http_addr, status,
		        approved_at, approved_by, last_heartbeat_at, version, metadata, created_at, updated_at
		 FROM edge_nodes WHERE status = 'active' AND (last_heartbeat_at IS NULL OR last_heartbeat_at < $1)
		 ORDER BY last_heartbeat_at ASC`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list stale edge nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*EdgeNode
	for rows.Next() {
		n := &EdgeNode{}
		if err := rows.Scan(&n.ID, &n.NodeID, &n.Name, &n.Region, &n.PublicAddr, &n.HTTPAddr, &n.Status,
			&n.ApprovedAt, &n.ApprovedBy, &n.LastHeartbeatAt, &n.Version, &n.Metadata, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan stale edge node: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
