package database

import (
	"context"
	"fmt"

	"github.com/mephistofox/fxtunnel/internal/server/database/sqlc"
)

// SessionRepository handles session database operations using PostgreSQL via sqlc.
type SessionRepository struct {
	q *sqlc.Queries
}

// sqlcSessionToDomain converts a sqlc.Session to a domain Session.
func sqlcSessionToDomain(s sqlc.Session) *Session {
	return &Session{
		ID:               s.ID,
		UserID:           s.UserID,
		RefreshTokenHash: s.RefreshTokenHash,
		UserAgent:        textToString(s.UserAgent),
		IPAddress:        textToString(s.IpAddress),
		ExpiresAt:        tsToTime(s.ExpiresAt),
		CreatedAt:        tsToTime(s.CreatedAt),
	}
}

// Create creates a new session.
func (r *SessionRepository) Create(session *Session) error {
	ctx := context.Background()
	row, err := r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:           session.UserID,
		RefreshTokenHash: session.RefreshTokenHash,
		UserAgent:        stringToPgtext(session.UserAgent),
		IpAddress:        stringToPgtext(session.IPAddress),
		ExpiresAt:        timeToPgtz(session.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	session.ID = row.ID
	session.CreatedAt = tsToTime(row.CreatedAt)
	return nil
}

// GetByTokenHash retrieves a session by refresh token hash.
func (r *SessionRepository) GetByTokenHash(tokenHash string) (*Session, error) {
	ctx := context.Background()
	s, err := r.q.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session by token hash: %w", err)
	}
	return sqlcSessionToDomain(s), nil
}

// GetByUserID retrieves all sessions for a user.
func (r *SessionRepository) GetByUserID(userID int64) ([]*Session, error) {
	ctx := context.Background()
	rows, err := r.q.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get sessions by user id: %w", err)
	}
	sessions := make([]*Session, 0, len(rows))
	for _, s := range rows {
		sessions = append(sessions, sqlcSessionToDomain(s))
	}
	return sessions, nil
}

// Delete deletes a session by ID.
func (r *SessionRepository) Delete(id int64) error {
	ctx := context.Background()
	err := r.q.DeleteSession(ctx, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteByTokenHash deletes a session by refresh token hash.
func (r *SessionRepository) DeleteByTokenHash(tokenHash string) error {
	ctx := context.Background()
	err := r.q.DeleteSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session by token hash: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all sessions for a user.
func (r *SessionRepository) DeleteByUserID(userID int64) error {
	ctx := context.Background()
	err := r.q.DeleteSessionsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("delete sessions by user id: %w", err)
	}
	return nil
}

// DeleteExpired deletes all expired sessions.
func (r *SessionRepository) DeleteExpired() (int64, error) {
	ctx := context.Background()
	count, err := r.q.DeleteExpiredSessions(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return count, nil
}
