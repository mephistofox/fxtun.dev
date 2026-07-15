package database

import (
	"context"
	"fmt"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/database/sqlc"
)

// UserHistoryRepository handles user history database operations using PostgreSQL via sqlc.
type UserHistoryRepository struct {
	q *sqlc.Queries
}

// sqlcHistoryToDomain converts a sqlc.UserHistory to a domain UserHistoryEntry.
func sqlcHistoryToDomain(h sqlc.UserHistory) *UserHistoryEntry {
	return &UserHistoryEntry{
		ID:             h.ID,
		UserID:         h.UserID,
		BundleName:     textToString(h.BundleName),
		TunnelType:     h.TunnelType,
		LocalPort:      int(h.LocalPort),
		RemoteAddr:     textToString(h.RemoteAddr),
		URL:            textToString(h.Url),
		ConnectedAt:    tsToTime(h.ConnectedAt),
		DisconnectedAt: tsToTimePtr(h.DisconnectedAt),
		BytesSent:      h.BytesSent,
		BytesReceived:  h.BytesReceived,
	}
}

// Create creates a new history entry.
func (r *UserHistoryRepository) Create(entry *UserHistoryEntry) error {
	ctx := context.Background()
	id, err := r.q.CreateHistoryEntry(ctx, sqlc.CreateHistoryEntryParams{
		UserID:         entry.UserID,
		BundleName:     stringToPgtext(entry.BundleName),
		TunnelType:     entry.TunnelType,
		LocalPort:      int32(entry.LocalPort),
		RemoteAddr:     stringToPgtext(entry.RemoteAddr),
		Url:            stringToPgtext(entry.URL),
		ConnectedAt:    timeToPgtz(entry.ConnectedAt),
		DisconnectedAt: timePtrToPgtz(entry.DisconnectedAt),
		BytesSent:      entry.BytesSent,
		BytesReceived:  entry.BytesReceived,
	})
	if err != nil {
		return fmt.Errorf("create history entry: %w", err)
	}
	entry.ID = id
	return nil
}

// Update updates a history entry (typically to set disconnected_at and byte counts).
func (r *UserHistoryRepository) Update(entry *UserHistoryEntry) error {
	ctx := context.Background()
	err := r.q.UpdateHistoryEntry(ctx, sqlc.UpdateHistoryEntryParams{
		ID:             entry.ID,
		UserID:         entry.UserID,
		DisconnectedAt: timePtrToPgtz(entry.DisconnectedAt),
		BytesSent:      entry.BytesSent,
		BytesReceived:  entry.BytesReceived,
	})
	if err != nil {
		return fmt.Errorf("update history entry: %w", err)
	}
	return nil
}

// GetByID retrieves a history entry by ID and user ID.
func (r *UserHistoryRepository) GetByID(id, userID int64) (*UserHistoryEntry, error) {
	ctx := context.Background()
	h, err := r.q.GetHistoryEntryByID(ctx, sqlc.GetHistoryEntryByIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrHistoryNotFound
		}
		return nil, fmt.Errorf("get history entry by id: %w", err)
	}
	return sqlcHistoryToDomain(h), nil
}

// GetByUserID retrieves history entries for a user with pagination.
func (r *UserHistoryRepository) GetByUserID(userID int64, limit, offset int) ([]*UserHistoryEntry, error) {
	ctx := context.Background()
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := r.q.ListHistoryByUserID(ctx, sqlc.ListHistoryByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("get history by user id: %w", err)
	}
	entries := make([]*UserHistoryEntry, 0, len(rows))
	for _, h := range rows {
		entries = append(entries, sqlcHistoryToDomain(h))
	}
	return entries, nil
}

// GetRecent retrieves the most recent history entries for a user.
func (r *UserHistoryRepository) GetRecent(userID int64, limit int) ([]*UserHistoryEntry, error) {
	return r.GetByUserID(userID, limit, 0)
}

// AddBulk adds multiple history entries.
func (r *UserHistoryRepository) AddBulk(userID int64, entries []*UserHistoryEntry) error {
	ctx := context.Background()
	for _, entry := range entries {
		entry.UserID = userID
		id, err := r.q.CreateHistoryEntry(ctx, sqlc.CreateHistoryEntryParams{
			UserID:         entry.UserID,
			BundleName:     stringToPgtext(entry.BundleName),
			TunnelType:     entry.TunnelType,
			LocalPort:      int32(entry.LocalPort),
			RemoteAddr:     stringToPgtext(entry.RemoteAddr),
			Url:            stringToPgtext(entry.URL),
			ConnectedAt:    timeToPgtz(entry.ConnectedAt),
			DisconnectedAt: timePtrToPgtz(entry.DisconnectedAt),
			BytesSent:      entry.BytesSent,
			BytesReceived:  entry.BytesReceived,
		})
		if err != nil {
			return fmt.Errorf("insert history entry: %w", err)
		}
		entry.ID = id
	}
	return nil
}

// Clear deletes all history entries for a user.
func (r *UserHistoryRepository) Clear(userID int64) error {
	ctx := context.Background()
	err := r.q.ClearHistory(ctx, userID)
	if err != nil {
		return fmt.Errorf("clear history: %w", err)
	}
	return nil
}

// Count returns the total number of history entries for a user.
func (r *UserHistoryRepository) Count(userID int64) (int, error) {
	ctx := context.Background()
	count, err := r.q.CountHistoryByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count history: %w", err)
	}
	return int(count), nil
}

// GetStats returns aggregated statistics for a user.
func (r *UserHistoryRepository) GetStats(userID int64) (*HistoryStats, error) {
	ctx := context.Background()
	row, err := r.q.GetHistoryStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get history stats: %w", err)
	}

	stats := &HistoryStats{
		TotalConnections: int(row.TotalConnections),
	}

	// TotalBytesSent and TotalBytesReceived come as interface{} from COALESCE(SUM(...))
	if v, ok := row.TotalBytesSent.(int64); ok {
		stats.TotalBytesSent = v
	}
	if v, ok := row.TotalBytesReceived.(int64); ok {
		stats.TotalBytesReceived = v
	}

	return stats, nil
}

// DeleteOlderThan deletes history entries older than the given time.
func (r *UserHistoryRepository) DeleteOlderThan(userID int64, before time.Time) (int64, error) {
	ctx := context.Background()
	count, err := r.q.DeleteHistoryOlderThan(ctx, sqlc.DeleteHistoryOlderThanParams{
		UserID:      userID,
		ConnectedAt: timeToPgtz(before),
	})
	if err != nil {
		return 0, fmt.Errorf("delete old history: %w", err)
	}
	return count, nil
}
