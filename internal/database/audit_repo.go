package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// AuditRepository handles audit log database operations
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Log creates a new audit log entry
func (r *AuditRepository) Log(userID *int64, action string, details map[string]interface{}, ipAddress string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal details: %w", err)
	}

	query := `
		INSERT INTO audit_logs (user_id, action, details, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(query, userID, action, string(detailsJSON), ipAddress, time.Now())
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

// GetByUserID retrieves audit logs for a user with pagination
func (r *AuditRepository) GetByUserID(userID int64, limit, offset int) ([]*AuditLog, int, error) {
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE user_id = ?`
	var total int
	if err := r.db.QueryRow(countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	query := `
		SELECT id, user_id, action, details, ip_address, created_at
		FROM audit_logs WHERE user_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`

	return r.queryAuditLogs(query, total, userID, limit, offset)
}

// List retrieves all audit logs with pagination
func (r *AuditRepository) List(limit, offset int) ([]*AuditLog, int, error) {
	countQuery := `SELECT COUNT(*) FROM audit_logs`
	var total int
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	query := `
		SELECT id, user_id, action, details, ip_address, created_at
		FROM audit_logs
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`

	return r.queryAuditLogs(query, total, limit, offset)
}

// ListByAction retrieves audit logs by action type with pagination
func (r *AuditRepository) ListByAction(action string, limit, offset int) ([]*AuditLog, int, error) {
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE action = ?`
	var total int
	if err := r.db.QueryRow(countQuery, action).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	query := `
		SELECT id, user_id, action, details, ip_address, created_at
		FROM audit_logs WHERE action = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`

	return r.queryAuditLogs(query, total, action, limit, offset)
}

// queryAuditLogs executes a query and returns audit logs
func (r *AuditRepository) queryAuditLogs(query string, total int, args ...interface{}) ([]*AuditLog, int, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		var userID sql.NullInt64
		var details, ipAddress sql.NullString

		if err := rows.Scan(
			&log.ID,
			&userID,
			&log.Action,
			&details,
			&ipAddress,
			&log.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}

		if userID.Valid {
			log.UserID = &userID.Int64
		}
		if ipAddress.Valid {
			log.IPAddress = ipAddress.String
		}
		if details.Valid && details.String != "" {
			if err := json.Unmarshal([]byte(details.String), &log.Details); err != nil {
				log.Details = map[string]interface{}{}
			}
		}

		logs = append(logs, log)
	}

	return logs, total, nil
}

// DeleteOlderThan deletes audit logs older than the specified duration
func (r *AuditRepository) DeleteOlderThan(duration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-duration)
	query := `DELETE FROM audit_logs WHERE created_at < ?`

	result, err := r.db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old audit logs: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rows, nil
}

// GetLatestByUserAndAction retrieves the latest audit log for a user and action
func (r *AuditRepository) GetLatestByUserAndAction(userID int64, action string) (*AuditLog, error) {
	query := `
		SELECT id, user_id, action, details, ip_address, created_at
		FROM audit_logs WHERE user_id = ? AND action = ?
		ORDER BY created_at DESC LIMIT 1
	`

	log := &AuditLog{}
	var uid sql.NullInt64
	var details, ipAddress sql.NullString

	err := r.db.QueryRow(query, userID, action).Scan(
		&log.ID,
		&uid,
		&log.Action,
		&details,
		&ipAddress,
		&log.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest audit log: %w", err)
	}

	if uid.Valid {
		log.UserID = &uid.Int64
	}
	if ipAddress.Valid {
		log.IPAddress = ipAddress.String
	}
	if details.Valid && details.String != "" {
		if err := json.Unmarshal([]byte(details.String), &log.Details); err != nil {
			log.Details = map[string]interface{}{}
		}
	}

	return log, nil
}
