package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditRepo_Log(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	err := db.Audit.Log(&user.ID, ActionLogin, map[string]interface{}{"browser": "chrome"}, "127.0.0.1")
	require.NoError(t, err)
}

func TestAuditRepo_LogNilUser(t *testing.T) {
	db := newTestDB(t)

	err := db.Audit.Log(nil, ActionRegister, nil, "10.0.0.1")
	require.NoError(t, err)
}

func TestAuditRepo_GetByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	require.NoError(t, db.Audit.Log(&user.ID, ActionLogin, nil, "127.0.0.1"))
	require.NoError(t, db.Audit.Log(&user.ID, ActionLogout, nil, "127.0.0.1"))

	logs, total, err := db.Audit.GetByUserID(user.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, logs, 2)
}

func TestAuditRepo_List(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	require.NoError(t, db.Audit.Log(&user.ID, ActionLogin, nil, "127.0.0.1"))
	require.NoError(t, db.Audit.Log(nil, ActionRegister, nil, "10.0.0.1"))

	logs, total, err := db.Audit.List(10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, logs, 2)
}

func TestAuditRepo_ListPagination(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 5; i++ {
		require.NoError(t, db.Audit.Log(nil, ActionLogin, nil, "127.0.0.1"))
	}

	logs, total, err := db.Audit.List(2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, logs, 2)
}

func TestAuditRepo_ListByAction(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Audit.Log(nil, ActionLogin, nil, "127.0.0.1"))
	require.NoError(t, db.Audit.Log(nil, ActionLogout, nil, "127.0.0.1"))
	require.NoError(t, db.Audit.Log(nil, ActionLogin, nil, "127.0.0.1"))

	logs, total, err := db.Audit.ListByAction(ActionLogin, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, logs, 2)
}

func TestAuditRepo_DeleteOlderThan(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Audit.Log(nil, ActionLogin, nil, "127.0.0.1"))

	// Delete logs older than 0 duration (i.e., all existing logs)
	count, err := db.Audit.DeleteOlderThan(0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestAuditRepo_GetLatestByUserAndAction(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	require.NoError(t, db.Audit.Log(&user.ID, ActionLogin, map[string]interface{}{"attempt": float64(1)}, "127.0.0.1"))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, db.Audit.Log(&user.ID, ActionLogin, map[string]interface{}{"attempt": float64(2)}, "10.0.0.1"))

	got, err := db.Audit.GetLatestByUserAndAction(user.ID, ActionLogin)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "10.0.0.1", got.IPAddress)
}

func TestAuditRepo_GetLatestByUserAndAction_NotFound(t *testing.T) {
	db := newTestDB(t)

	got, err := db.Audit.GetLatestByUserAndAction(999, ActionLogin)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestAuditRepo_Details(t *testing.T) {
	db := newTestDB(t)
	details := map[string]interface{}{"key": "value", "count": float64(42)}
	require.NoError(t, db.Audit.Log(nil, ActionLogin, details, "127.0.0.1"))

	logs, _, err := db.Audit.List(1, 0)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "value", logs[0].Details["key"])
	assert.Equal(t, float64(42), logs[0].Details["count"])
}
