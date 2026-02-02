package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepo_Create(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	assert.NotZero(t, user.ID)
	assert.NotZero(t, user.CreatedAt)
}

func TestUserRepo_CreateDuplicate(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "+1111111111")

	freePlan, _ := db.Plans.GetDefault()
	dup := &User{
		Phone:        "+1111111111",
		PasswordHash: "hash",
		IsActive:     true,
		PlanID:       freePlan.ID,
	}
	err := db.Users.Create(dup)
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestUserRepo_GetByID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	got, err := db.Users.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Phone, got.Phone)
	assert.Equal(t, user.DisplayName, got.DisplayName)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Users.GetByID(999)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_GetByPhone(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	got, err := db.Users.GetByPhone("+1111111111")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
}

func TestUserRepo_GetByPhone_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Users.GetByPhone("+0000000000")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_Update(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	user.DisplayName = "Updated"
	user.IsAdmin = true
	require.NoError(t, db.Users.Update(user))

	got, err := db.Users.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.DisplayName)
	assert.True(t, got.IsAdmin)
}

func TestUserRepo_Update_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Users.Update(&User{ID: 999, DisplayName: "x"})
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_UpdatePassword(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	require.NoError(t, db.Users.UpdatePassword(user.ID, "newhash"))

	got, err := db.Users.GetByID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "newhash", got.PasswordHash)
}

func TestUserRepo_UpdatePassword_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Users.UpdatePassword(999, "hash")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_UpdateLastLogin(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	require.NoError(t, db.Users.UpdateLastLogin(user.ID))

	got, err := db.Users.GetByID(user.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.LastLoginAt)
}

func TestUserRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	require.NoError(t, db.Users.Delete(user.ID))

	_, err := db.Users.GetByID(user.ID)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Users.Delete(999)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepo_List(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "+1111111111")
	createTestUser(t, db, "+2222222222")
	createTestUser(t, db, "+3333333333")

	users, total, err := db.Users.List(2, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, users, 2)

	users2, total2, err := db.Users.List(2, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total2)
	assert.Len(t, users2, 1)
}

func TestUserRepo_Count(t *testing.T) {
	db := newTestDB(t)
	createTestUser(t, db, "+1111111111")
	createTestUser(t, db, "+2222222222")

	count, err := db.Users.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
