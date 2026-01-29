package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestSession(t *testing.T, db *Database, userID int64, tokenHash string) *Session {
	t.Helper()
	s := &Session{
		UserID:           userID,
		RefreshTokenHash: tokenHash,
		UserAgent:        "TestAgent",
		IPAddress:        "127.0.0.1",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, db.Sessions.Create(s))
	return s
}

func TestSessionRepo_Create(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	s := createTestSession(t, db, user.ID, "hash1")

	assert.NotZero(t, s.ID)
	assert.NotZero(t, s.CreatedAt)
}

func TestSessionRepo_GetByTokenHash(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestSession(t, db, user.ID, "hash1")

	got, err := db.Sessions.GetByTokenHash("hash1")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.UserID)
	assert.Equal(t, "TestAgent", got.UserAgent)
}

func TestSessionRepo_GetByTokenHash_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Sessions.GetByTokenHash("nonexistent")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionRepo_GetByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestSession(t, db, user.ID, "hash1")
	createTestSession(t, db, user.ID, "hash2")

	sessions, err := db.Sessions.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestSessionRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	s := createTestSession(t, db, user.ID, "hash1")

	require.NoError(t, db.Sessions.Delete(s.ID))
	_, err := db.Sessions.GetByTokenHash("hash1")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionRepo_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Sessions.Delete(999)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionRepo_DeleteByTokenHash(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestSession(t, db, user.ID, "hash1")

	require.NoError(t, db.Sessions.DeleteByTokenHash("hash1"))
	_, err := db.Sessions.GetByTokenHash("hash1")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionRepo_DeleteByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestSession(t, db, user.ID, "hash1")
	createTestSession(t, db, user.ID, "hash2")

	require.NoError(t, db.Sessions.DeleteByUserID(user.ID))
	sessions, err := db.Sessions.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionRepo_DeleteExpired(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	// Create expired session
	expired := &Session{
		UserID:           user.ID,
		RefreshTokenHash: "expired_hash",
		ExpiresAt:        time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, db.Sessions.Create(expired))

	// Create valid session
	createTestSession(t, db, user.ID, "valid_hash")

	count, err := db.Sessions.DeleteExpired()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	sessions, err := db.Sessions.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}

func TestSessionRepo_CascadeDeleteUser(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestSession(t, db, user.ID, "hash1")

	require.NoError(t, db.Users.Delete(user.ID))

	sessions, err := db.Sessions.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}
