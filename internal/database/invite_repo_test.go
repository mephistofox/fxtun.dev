package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestInvite(t *testing.T, db *Database, code string, createdBy *int64) *InviteCode {
	t.Helper()
	inv := &InviteCode{
		Code:            code,
		CreatedByUserID: createdBy,
	}
	require.NoError(t, db.Invites.Create(inv))
	return inv
}

func TestInviteRepo_Create(t *testing.T) {
	db := newTestDB(t)
	inv := createTestInvite(t, db, "ABC123", nil)

	assert.NotZero(t, inv.ID)
	assert.NotZero(t, inv.CreatedAt)
}

func TestInviteRepo_CreateDuplicate(t *testing.T) {
	db := newTestDB(t)
	createTestInvite(t, db, "ABC123", nil)

	err := db.Invites.Create(&InviteCode{Code: "ABC123"})
	assert.ErrorIs(t, err, ErrInviteAlreadyExists)
}

func TestInviteRepo_GetByID(t *testing.T) {
	db := newTestDB(t)
	inv := createTestInvite(t, db, "ABC123", nil)

	got, err := db.Invites.GetByID(inv.ID)
	require.NoError(t, err)
	assert.Equal(t, "ABC123", got.Code)
}

func TestInviteRepo_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Invites.GetByID(999)
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestInviteRepo_GetByCode(t *testing.T) {
	db := newTestDB(t)
	createTestInvite(t, db, "ABC123", nil)

	got, err := db.Invites.GetByCode("ABC123")
	require.NoError(t, err)
	assert.Equal(t, "ABC123", got.Code)
}

func TestInviteRepo_GetByCode_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Invites.GetByCode("NONEXISTENT")
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestInviteRepo_Use(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestInvite(t, db, "ABC123", nil)

	require.NoError(t, db.Invites.Use("ABC123", user.ID))

	got, err := db.Invites.GetByCode("ABC123")
	require.NoError(t, err)
	assert.NotNil(t, got.UsedByUserID)
	assert.Equal(t, user.ID, *got.UsedByUserID)
	assert.NotNil(t, got.UsedAt)
}

func TestInviteRepo_UseAlreadyUsed(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestInvite(t, db, "ABC123", nil)

	require.NoError(t, db.Invites.Use("ABC123", user.ID))
	err := db.Invites.Use("ABC123", user.ID)
	assert.ErrorIs(t, err, ErrInviteAlreadyUsed)
}

func TestInviteRepo_UseExpired(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	expired := time.Now().Add(-1 * time.Hour)
	inv := &InviteCode{Code: "EXPIRED", ExpiresAt: &expired}
	require.NoError(t, db.Invites.Create(inv))

	err := db.Invites.Use("EXPIRED", user.ID)
	assert.ErrorIs(t, err, ErrInviteExpired)
}

func TestInviteRepo_UseNotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Invites.Use("NONEXISTENT", 1)
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestInviteRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	inv := createTestInvite(t, db, "ABC123", nil)

	require.NoError(t, db.Invites.Delete(inv.ID))
	_, err := db.Invites.GetByID(inv.ID)
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestInviteRepo_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Invites.Delete(999)
	assert.ErrorIs(t, err, ErrInviteNotFound)
}

func TestInviteRepo_List(t *testing.T) {
	db := newTestDB(t)
	createTestInvite(t, db, "A", nil)
	createTestInvite(t, db, "B", nil)
	createTestInvite(t, db, "C", nil)

	invites, total, err := db.Invites.List(2, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, invites, 2)
}

func TestInviteRepo_ListUnused(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestInvite(t, db, "USED", nil)
	createTestInvite(t, db, "UNUSED", nil)
	require.NoError(t, db.Invites.Use("USED", user.ID))

	invites, total, err := db.Invites.ListUnused(10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, invites, 1)
	assert.Equal(t, "UNUSED", invites[0].Code)
}

func TestInviteRepo_DeleteExpired(t *testing.T) {
	db := newTestDB(t)
	expired := time.Now().Add(-1 * time.Hour)
	require.NoError(t, db.Invites.Create(&InviteCode{Code: "EXP", ExpiresAt: &expired}))
	createTestInvite(t, db, "VALID", nil)

	count, err := db.Invites.DeleteExpired()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	_, total, err := db.Invites.List(10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestInviteRepo_IsValid(t *testing.T) {
	db := newTestDB(t)
	createTestInvite(t, db, "VALID", nil)

	valid, err := db.Invites.IsValid("VALID")
	require.NoError(t, err)
	assert.True(t, valid)

	valid, err = db.Invites.IsValid("NONEXISTENT")
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestInviteRepo_IsValid_Expired(t *testing.T) {
	db := newTestDB(t)
	expired := time.Now().Add(-1 * time.Hour)
	require.NoError(t, db.Invites.Create(&InviteCode{Code: "EXP", ExpiresAt: &expired}))

	valid, err := db.Invites.IsValid("EXP")
	require.NoError(t, err)
	assert.False(t, valid)
}
