package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

// SessionRepository handles session database operations
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(session *Session) error {
	query := `
		INSERT INTO sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		session.UserID,
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	session.ID = id
	session.CreatedAt = now
	return nil
}

// GetByTokenHash retrieves a session by refresh token hash
func (r *SessionRepository) GetByTokenHash(tokenHash string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
		FROM sessions WHERE refresh_token_hash = ?
	`

	session := &Session{}
	var userAgent, ipAddress sql.NullString

	err := r.db.QueryRow(query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&userAgent,
		&ipAddress,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, notFoundOrError(err, ErrSessionNotFound, "get session by token hash")
	}

	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}

	return session, nil
}

// GetByUserID retrieves all sessions for a user
func (r *SessionRepository) GetByUserID(userID int64) ([]*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
		FROM sessions WHERE user_id = ? ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get sessions by user id: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		var userAgent, ipAddress sql.NullString

		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenHash,
			&userAgent,
			&ipAddress,
			&session.ExpiresAt,
			&session.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		if userAgent.Valid {
			session.UserAgent = userAgent.String
		}
		if ipAddress.Valid {
			session.IPAddress = ipAddress.String
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Delete deletes a session by ID
func (r *SessionRepository) Delete(id int64) error {
	query := `DELETE FROM sessions WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteByTokenHash deletes a session by refresh token hash
func (r *SessionRepository) DeleteByTokenHash(tokenHash string) error {
	query := `DELETE FROM sessions WHERE refresh_token_hash = ?`

	_, err := r.db.Exec(query, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session by token hash: %w", err)
	}

	return nil
}

// DeleteByUserID deletes all sessions for a user
func (r *SessionRepository) DeleteByUserID(userID int64) error {
	query := `DELETE FROM sessions WHERE user_id = ?`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("delete sessions by user id: %w", err)
	}

	return nil
}

// DeleteExpired deletes all expired sessions
func (r *SessionRepository) DeleteExpired() (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < ?`

	result, err := r.db.Exec(query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rows, nil
}
