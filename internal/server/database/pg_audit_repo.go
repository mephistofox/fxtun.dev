package database

import (
	"context"
	"fmt"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// AuditRepository handles audit log database operations using PostgreSQL via sqlc.
type AuditRepository struct {
	q *sqlc.Queries
}

// sqlcAuditToDomain converts a sqlc.AuditLog to a domain AuditLog.
func sqlcAuditToDomain(a sqlc.AuditLog) *AuditLog {
	return &AuditLog{
		ID:        a.ID,
		UserID:    int8ToInt64Ptr(a.UserID),
		Action:    a.Action,
		Details:   jsonToMap(a.Details),
		IPAddress: textToString(a.IpAddress),
		CreatedAt: tsToTime(a.CreatedAt),
	}
}

// Log creates a new audit log entry.
func (r *AuditRepository) Log(userID *int64, action string, details map[string]interface{}, ipAddress string) error {
	ctx := context.Background()
	err := r.q.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		UserID:    int64PtrToPgint8(userID),
		Action:    action,
		Details:   mapToJSON(details),
		IpAddress: stringToPgtext(ipAddress),
	})
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// GetByUserID retrieves audit logs for a user with pagination.
func (r *AuditRepository) GetByUserID(userID int64, limit, offset int) ([]*AuditLog, int, error) {
	ctx := context.Background()

	count, err := r.q.CountAuditLogsByUserID(ctx, int64ToPgint8(userID))
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs by user id: %w", err)
	}

	rows, err := r.q.ListAuditLogsByUserID(ctx, sqlc.ListAuditLogsByUserIDParams{
		UserID: int64ToPgint8(userID),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs by user id: %w", err)
	}

	logs := make([]*AuditLog, 0, len(rows))
	for _, a := range rows {
		logs = append(logs, sqlcAuditToDomain(a))
	}
	return logs, int(count), nil
}

// List retrieves all audit logs with pagination.
func (r *AuditRepository) List(limit, offset int) ([]*AuditLog, int, error) {
	ctx := context.Background()

	count, err := r.q.CountAuditLogs(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := r.q.ListAuditLogs(ctx, sqlc.ListAuditLogsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}

	logs := make([]*AuditLog, 0, len(rows))
	for _, a := range rows {
		logs = append(logs, sqlcAuditToDomain(a))
	}
	return logs, int(count), nil
}

// ListByAction retrieves audit logs filtered by action with pagination.
func (r *AuditRepository) ListByAction(action string, limit, offset int) ([]*AuditLog, int, error) {
	ctx := context.Background()

	count, err := r.q.CountAuditLogsByAction(ctx, action)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs by action: %w", err)
	}

	rows, err := r.q.ListAuditLogsByAction(ctx, sqlc.ListAuditLogsByActionParams{
		Action: action,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs by action: %w", err)
	}

	logs := make([]*AuditLog, 0, len(rows))
	for _, a := range rows {
		logs = append(logs, sqlcAuditToDomain(a))
	}
	return logs, int(count), nil
}

// DeleteOlderThan deletes audit logs older than the given duration.
func (r *AuditRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	ctx := context.Background()
	cutoff := time.Now().Add(-duration)
	count, err := r.q.DeleteAuditLogsOlderThan(ctx, timeToPgtz(cutoff))
	if err != nil {
		return 0, fmt.Errorf("delete old audit logs: %w", err)
	}
	return count, nil
}

// GetLatestByUserAndAction retrieves the most recent audit log for a user and action.
func (r *AuditRepository) GetLatestByUserAndAction(userID int64, action string) (*AuditLog, error) {
	ctx := context.Background()
	a, err := r.q.GetLatestAuditLogByUserAndAction(ctx, sqlc.GetLatestAuditLogByUserAndActionParams{
		UserID: int64ToPgint8(userID),
		Action: action,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest audit log: %w", err)
	}
	return sqlcAuditToDomain(a), nil
}
