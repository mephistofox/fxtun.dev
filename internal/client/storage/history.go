package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// HistoryEntry represents a connection history record
type HistoryEntry struct {
	ID             int64      `json:"id"`
	BundleID       *int64     `json:"bundle_id,omitempty"`
	BundleName     string     `json:"bundle_name,omitempty"`
	TunnelType     string     `json:"tunnel_type"`
	LocalPort      int        `json:"local_port"`
	RemoteAddr     string     `json:"remote_addr,omitempty"`
	URL            string     `json:"url,omitempty"`
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
	BytesSent      int64      `json:"bytes_sent"`
	BytesReceived  int64      `json:"bytes_received"`
}

// HistoryRepository provides operations for connection history
type HistoryRepository struct {
	db *Database
}

// NewHistoryRepository creates a new history repository
func NewHistoryRepository(db *Database) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// List returns history entries with pagination
func (r *HistoryRepository) List(limit, offset int) ([]HistoryEntry, int, error) {
	// Get total count
	var total int
	if err := r.db.db.QueryRow("SELECT COUNT(*) FROM history").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count history: %w", err)
	}

	rows, err := r.db.db.Query(`
		SELECT id, bundle_id, bundle_name, tunnel_type, local_port, remote_addr, url,
		       connected_at, disconnected_at, bytes_sent, bytes_received
		FROM history
		ORDER BY connected_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var bundleID sql.NullInt64
		var bundleName, remoteAddr, url sql.NullString
		var disconnectedAt sql.NullTime

		if err := rows.Scan(&e.ID, &bundleID, &bundleName, &e.TunnelType, &e.LocalPort,
			&remoteAddr, &url, &e.ConnectedAt, &disconnectedAt, &e.BytesSent, &e.BytesReceived); err != nil {
			return nil, 0, fmt.Errorf("scan history: %w", err)
		}

		if bundleID.Valid {
			e.BundleID = &bundleID.Int64
		}
		if bundleName.Valid {
			e.BundleName = bundleName.String
		}
		if remoteAddr.Valid {
			e.RemoteAddr = remoteAddr.String
		}
		if url.Valid {
			e.URL = url.String
		}
		if disconnectedAt.Valid {
			e.DisconnectedAt = &disconnectedAt.Time
		}

		entries = append(entries, e)
	}

	return entries, total, rows.Err()
}

// RecordConnect records a new connection
func (r *HistoryRepository) RecordConnect(entry *HistoryEntry) error {
	result, err := r.db.db.Exec(`
		INSERT INTO history (bundle_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, nullInt64Ptr(entry.BundleID), nullString(entry.BundleName), entry.TunnelType,
		entry.LocalPort, nullString(entry.RemoteAddr), nullString(entry.URL), entry.ConnectedAt)

	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	entry.ID = id
	return nil
}

// RecordDisconnect updates a history entry with disconnect time and traffic stats
func (r *HistoryRepository) RecordDisconnect(id int64, bytesSent, bytesReceived int64) error {
	now := time.Now()
	_, err := r.db.db.Exec(`
		UPDATE history
		SET disconnected_at = ?, bytes_sent = ?, bytes_received = ?
		WHERE id = ?
	`, now, bytesSent, bytesReceived, id)

	if err != nil {
		return fmt.Errorf("update history: %w", err)
	}
	return nil
}

// Clear deletes all history entries
func (r *HistoryRepository) Clear() error {
	_, err := r.db.db.Exec("DELETE FROM history")
	if err != nil {
		return fmt.Errorf("clear history: %w", err)
	}
	return nil
}

// GetRecent returns the most recent history entries
func (r *HistoryRepository) GetRecent(limit int) ([]HistoryEntry, error) {
	entries, _, err := r.List(limit, 0)
	return entries, err
}

// Helper function
func nullInt64Ptr(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
