package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestToken(t *testing.T, db *Database, userID int64, name, hash string) *APIToken {
	t.Helper()
	token := &APIToken{
		UserID:            userID,
		TokenHash:         hash,
		Name:              name,
		AllowedSubdomains: []string{"*"},
		MaxTunnels:        10,
	}
	require.NoError(t, db.Tokens.Create(token))
	return token
}

func TestTokenRepo_Create(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	token := createTestToken(t, db, user.ID, "test", "tokenhash1")

	assert.NotZero(t, token.ID)
	assert.NotZero(t, token.CreatedAt)
}

func TestTokenRepo_GetByID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	token := createTestToken(t, db, user.ID, "test", "tokenhash1")

	got, err := db.Tokens.GetByID(token.ID)
	require.NoError(t, err)
	assert.Equal(t, "test", got.Name)
	assert.Equal(t, []string{"*"}, got.AllowedSubdomains)
}

func TestTokenRepo_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Tokens.GetByID(999)
	assert.ErrorIs(t, err, ErrTokenNotFound)
}

func TestTokenRepo_GetByTokenHash(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestToken(t, db, user.ID, "test", "tokenhash1")

	got, err := db.Tokens.GetByTokenHash("tokenhash1")
	require.NoError(t, err)
	assert.Equal(t, "test", got.Name)
}

func TestTokenRepo_GetByTokenHash_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Tokens.GetByTokenHash("nonexistent")
	assert.ErrorIs(t, err, ErrTokenNotFound)
}

func TestTokenRepo_GetByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestToken(t, db, user.ID, "t1", "hash1")
	createTestToken(t, db, user.ID, "t2", "hash2")

	tokens, err := db.Tokens.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
}

func TestTokenRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	token := createTestToken(t, db, user.ID, "test", "hash1")

	require.NoError(t, db.Tokens.Delete(token.ID))
	_, err := db.Tokens.GetByID(token.ID)
	assert.ErrorIs(t, err, ErrTokenNotFound)
}

func TestTokenRepo_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Tokens.Delete(999)
	assert.ErrorIs(t, err, ErrTokenNotFound)
}

func TestTokenRepo_DeleteByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestToken(t, db, user.ID, "t1", "hash1")
	createTestToken(t, db, user.ID, "t2", "hash2")

	require.NoError(t, db.Tokens.DeleteByUserID(user.ID))
	tokens, err := db.Tokens.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestTokenRepo_UpdateLastUsed(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	token := createTestToken(t, db, user.ID, "test", "hash1")

	require.NoError(t, db.Tokens.UpdateLastUsed(token.ID))

	got, err := db.Tokens.GetByID(token.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.LastUsedAt)
}

func TestTokenRepo_Count(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestToken(t, db, user.ID, "t1", "hash1")
	createTestToken(t, db, user.ID, "t2", "hash2")

	count, err := db.Tokens.Count(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestTokenRepo_CascadeDeleteUser(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestToken(t, db, user.ID, "test", "hash1")

	require.NoError(t, db.Users.Delete(user.ID))

	tokens, err := db.Tokens.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}
