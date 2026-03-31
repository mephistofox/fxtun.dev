package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTOTP(t *testing.T, db *Database, userID int64) *TOTPSecret {
	t.Helper()
	totp := &TOTPSecret{
		UserID:          userID,
		SecretEncrypted: "encrypted_secret",
		BackupCodes:     []string{"code1", "code2"},
	}
	require.NoError(t, db.TOTP.Create(totp))
	return totp
}

func TestTOTPRepo_Create(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	totp := createTestTOTP(t, db, user.ID)

	assert.NotZero(t, totp.ID)
	assert.NotZero(t, totp.CreatedAt)
}

func TestTOTPRepo_CreateDuplicate(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestTOTP(t, db, user.ID)

	err := db.TOTP.Create(&TOTPSecret{UserID: user.ID, SecretEncrypted: "s", BackupCodes: []string{}})
	assert.Error(t, err)
}

func TestTOTPRepo_GetByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestTOTP(t, db, user.ID)

	got, err := db.TOTP.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "encrypted_secret", got.SecretEncrypted)
	assert.Equal(t, []string{"code1", "code2"}, got.BackupCodes)
}

func TestTOTPRepo_GetByUserID_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.TOTP.GetByUserID(999)
	assert.ErrorIs(t, err, ErrTOTPNotFound)
}

func TestTOTPRepo_Update(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	totp := createTestTOTP(t, db, user.ID)

	totp.SecretEncrypted = "new_secret"
	totp.IsEnabled = true
	totp.BackupCodes = []string{"new1"}
	require.NoError(t, db.TOTP.Update(totp))

	got, err := db.TOTP.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, "new_secret", got.SecretEncrypted)
	assert.True(t, got.IsEnabled)
	assert.Equal(t, []string{"new1"}, got.BackupCodes)
}

func TestTOTPRepo_Enable(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestTOTP(t, db, user.ID)

	require.NoError(t, db.TOTP.Enable(user.ID))

	enabled, err := db.TOTP.IsEnabled(user.ID)
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestTOTPRepo_Enable_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.TOTP.Enable(999)
	assert.ErrorIs(t, err, ErrTOTPNotFound)
}

func TestTOTPRepo_Disable(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestTOTP(t, db, user.ID)
	require.NoError(t, db.TOTP.Enable(user.ID))

	require.NoError(t, db.TOTP.Disable(user.ID))

	enabled, err := db.TOTP.IsEnabled(user.ID)
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestTOTPRepo_Disable_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.TOTP.Disable(999)
	assert.ErrorIs(t, err, ErrTOTPNotFound)
}

func TestTOTPRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestTOTP(t, db, user.ID)

	require.NoError(t, db.TOTP.Delete(user.ID))

	_, err := db.TOTP.GetByUserID(user.ID)
	assert.ErrorIs(t, err, ErrTOTPNotFound)
}

func TestTOTPRepo_IsEnabled_NoRecord(t *testing.T) {
	db := newTestDB(t)

	enabled, err := db.TOTP.IsEnabled(999)
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestTOTPRepo_UpdateBackupCodes(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestTOTP(t, db, user.ID)

	require.NoError(t, db.TOTP.UpdateBackupCodes(user.ID, []string{"new1", "new2", "new3"}))

	got, err := db.TOTP.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"new1", "new2", "new3"}, got.BackupCodes)
}

func TestTOTPRepo_CascadeDeleteUser(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestTOTP(t, db, user.ID)

	require.NoError(t, db.Users.Delete(user.ID))

	_, err := db.TOTP.GetByUserID(user.ID)
	assert.ErrorIs(t, err, ErrTOTPNotFound)
}
