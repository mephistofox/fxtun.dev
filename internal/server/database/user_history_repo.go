package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrHistoryNotFound = errors.New("history entry not found")

// UserHistoryRepository handles user history database operations
type UserHistoryRepository struct {
	db *sql.DB
}

// NewUserHistoryRepository creates a new user history repository
func NewUserHistoryRepository(db *sql.DB) *UserHistoryRepository {
	return &UserHistoryRepository{db: db}
}

// Create creates a new history entry
func (r *UserHistoryRepository) Create(entry *UserHistoryEntry) error {
	query := `
		INSERT INTO user_history (user_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at, disconnected_at, bytes_sent, bytes_received)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		entry.UserID,
		entry.BundleName,
		entry.TunnelType,
		entry.LocalPort,
		entry.RemoteAddr,
		entry.URL,
		entry.ConnectedAt,
		entry.DisconnectedAt,
		entry.BytesSent,
		entry.BytesReceived,
	)
	if err != nil {
		return fmt.Errorf("create user history entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	entry.ID = id
	return nil
}

// Update updates a history entry (typically to set disconnected_at and byte counts)
func (r *UserHistoryRepository) Update(entry *UserHistoryEntry) error {
	query := `
		UPDATE user_history
		SET disconnected_at = ?, bytes_sent = ?, bytes_received = ?
		WHERE id = ? AND user_id = ?
	`

	result, err := r.db.Exec(query,
		entry.DisconnectedAt,
		entry.BytesSent,
		entry.BytesReceived,
		entry.ID,
		entry.UserID,
	)
	if err != nil {
		return fmt.Errorf("update user history entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrHistoryNotFound
	}

	return nil
}

// GetByID retrieves a history entry by ID
func (r *UserHistoryRepository) GetByID(id, userID int64) (*UserHistoryEntry, error) {
	query := `
		SELECT id, user_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at, disconnected_at, bytes_sent, bytes_received
		FROM user_history WHERE id = ? AND user_id = ?
	`

	entry := &UserHistoryEntry{}
	var bundleName, remoteAddr, url sql.NullString
	var disconnectedAt sql.NullTime

	err := r.db.QueryRow(query, id, userID).Scan(
		&entry.ID,
		&entry.UserID,
		&bundleName,
		&entry.TunnelType,
		&entry.LocalPort,
		&remoteAddr,
		&url,
		&entry.ConnectedAt,
		&disconnectedAt,
		&entry.BytesSent,
		&entry.BytesReceived,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrHistoryNotFound
		}
		return nil, fmt.Errorf("get user history entry: %w", err)
	}

	if bundleName.Valid {
		entry.BundleName = bundleName.String
	}
	if remoteAddr.Valid {
		entry.RemoteAddr = remoteAddr.String
	}
	if url.Valid {
		entry.URL = url.String
	}
	if disconnectedAt.Valid {
		entry.DisconnectedAt = &disconnectedAt.Time
	}

	return entry, nil
}

// GetByUserID retrieves history entries for a user with pagination
func (r *UserHistoryRepository) GetByUserID(userID int64, limit, offset int) ([]*UserHistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, user_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at, disconnected_at, bytes_sent, bytes_received
		FROM user_history WHERE user_id = ? ORDER BY connected_at DESC LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get user history entries: %w", err)
	}
	defer rows.Close()

	var entries []*UserHistoryEntry
	for rows.Next() {
		entry := &UserHistoryEntry{}
		var bundleName, remoteAddr, url sql.NullString
		var disconnectedAt sql.NullTime

		if err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&bundleName,
			&entry.TunnelType,
			&entry.LocalPort,
			&remoteAddr,
			&url,
			&entry.ConnectedAt,
			&disconnectedAt,
			&entry.BytesSent,
			&entry.BytesReceived,
		); err != nil {
			return nil, fmt.Errorf("scan user history entry: %w", err)
		}

		if bundleName.Valid {
			entry.BundleName = bundleName.String
		}
		if remoteAddr.Valid {
			entry.RemoteAddr = remoteAddr.String
		}
		if url.Valid {
			entry.URL = url.String
		}
		if disconnectedAt.Valid {
			entry.DisconnectedAt = &disconnectedAt.Time
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetRecent retrieves the most recent history entries for a user
func (r *UserHistoryRepository) GetRecent(userID int64, limit int) ([]*UserHistoryEntry, error) {
	return r.GetByUserID(userID, limit, 0)
}

// AddBulk adds multiple history entries in a transaction
func (r *UserHistoryRepository) AddBulk(userID int64, entries []*UserHistoryEntry) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `
		INSERT INTO user_history (user_id, bundle_name, tunnel_type, local_port, remote_addr, url, connected_at, disconnected_at, bytes_sent, bytes_received)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		entry.UserID = userID
		result, err := stmt.Exec(
			entry.UserID,
			entry.BundleName,
			entry.TunnelType,
			entry.LocalPort,
			entry.RemoteAddr,
			entry.URL,
			entry.ConnectedAt,
			entry.DisconnectedAt,
			entry.BytesSent,
			entry.BytesReceived,
		)
		if err != nil {
			return fmt.Errorf("insert history entry: %w", err)
		}
		id, _ := result.LastInsertId()
		entry.ID = id
	}

	return tx.Commit()
}

// Clear deletes all history entries for a user
func (r *UserHistoryRepository) Clear(userID int64) error {
	query := `DELETE FROM user_history WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("clear user history: %w", err)
	}
	return nil
}

// Count returns the total number of history entries for a user
func (r *UserHistoryRepository) Count(userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM user_history WHERE user_id = ?`
	var count int
	if err := r.db.QueryRow(query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count user history: %w", err)
	}
	return count, nil
}

// GetStats returns aggregated statistics for a user
func (r *UserHistoryRepository) GetStats(userID int64) (*HistoryStats, error) {
	query := `
		SELECT
			COUNT(*) as total_connections,
			COALESCE(SUM(bytes_sent), 0) as total_bytes_sent,
			COALESCE(SUM(bytes_received), 0) as total_bytes_received
		FROM user_history WHERE user_id = ?
	`

	stats := &HistoryStats{}
	err := r.db.QueryRow(query, userID).Scan(
		&stats.TotalConnections,
		&stats.TotalBytesSent,
		&stats.TotalBytesReceived,
	)
	if err != nil {
		return nil, fmt.Errorf("get history stats: %w", err)
	}

	return stats, nil
}

// HistoryStats represents aggregated history statistics
type HistoryStats struct {
	TotalConnections   int   `json:"total_connections"`
	TotalBytesSent     int64 `json:"total_bytes_sent"`
	TotalBytesReceived int64 `json:"total_bytes_received"`
}

// DeleteOlderThan deletes history entries older than the given time
func (r *UserHistoryRepository) DeleteOlderThan(userID int64, before time.Time) (int64, error) {
	query := `DELETE FROM user_history WHERE user_id = ? AND connected_at < ?`
	result, err := r.db.Exec(query, userID, before)
	if err != nil {
		return 0, fmt.Errorf("delete old history: %w", err)
	}
	count, _ := result.RowsAffected()
	return count, nil
}
