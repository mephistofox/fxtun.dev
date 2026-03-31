package database

import (
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := New(dbPath, zerolog.Nop())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func createTestUser(t *testing.T, db *Database, phone string) *User {
	t.Helper()
	freePlan, err := db.Plans.GetDefault()
	require.NoError(t, err)
	user := &User{
		Phone:        phone,
		PasswordHash: "$2a$12$fakehashfakehashfakehashfakehashfakehashfakehashfake",
		DisplayName:  "Test User",
		IsActive:     true,
		PlanID:       freePlan.ID,
	}
	require.NoError(t, db.Users.Create(user))
	return user
}
