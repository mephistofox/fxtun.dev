package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestDomain(t *testing.T, db *Database, userID int64, subdomain string) *ReservedDomain {
	t.Helper()
	d := &ReservedDomain{UserID: userID, Subdomain: subdomain}
	require.NoError(t, db.Domains.Create(d))
	return d
}

func TestDomainRepo_Create(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	d := createTestDomain(t, db, user.ID, "myapp")

	assert.NotZero(t, d.ID)
	assert.NotZero(t, d.CreatedAt)
}

func TestDomainRepo_CreateDuplicate(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestDomain(t, db, user.ID, "myapp")

	err := db.Domains.Create(&ReservedDomain{UserID: user.ID, Subdomain: "myapp"})
	assert.ErrorIs(t, err, ErrDomainAlreadyExists)
}

func TestDomainRepo_GetByID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	d := createTestDomain(t, db, user.ID, "myapp")

	got, err := db.Domains.GetByID(d.ID)
	require.NoError(t, err)
	assert.Equal(t, "myapp", got.Subdomain)
}

func TestDomainRepo_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Domains.GetByID(999)
	assert.ErrorIs(t, err, ErrDomainNotFound)
}

func TestDomainRepo_GetBySubdomain(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestDomain(t, db, user.ID, "myapp")

	got, err := db.Domains.GetBySubdomain("myapp")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.UserID)
}

func TestDomainRepo_GetBySubdomain_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.Domains.GetBySubdomain("nonexistent")
	assert.ErrorIs(t, err, ErrDomainNotFound)
}

func TestDomainRepo_GetByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestDomain(t, db, user.ID, "app1")
	createTestDomain(t, db, user.ID, "app2")

	domains, err := db.Domains.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Len(t, domains, 2)
}

func TestDomainRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	d := createTestDomain(t, db, user.ID, "myapp")

	require.NoError(t, db.Domains.Delete(d.ID))
	_, err := db.Domains.GetByID(d.ID)
	assert.ErrorIs(t, err, ErrDomainNotFound)
}

func TestDomainRepo_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	err := db.Domains.Delete(999)
	assert.ErrorIs(t, err, ErrDomainNotFound)
}

func TestDomainRepo_DeleteByUserID(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestDomain(t, db, user.ID, "app1")
	createTestDomain(t, db, user.ID, "app2")

	require.NoError(t, db.Domains.DeleteByUserID(user.ID))
	domains, err := db.Domains.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Empty(t, domains)
}

func TestDomainRepo_Count(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestDomain(t, db, user.ID, "app1")
	createTestDomain(t, db, user.ID, "app2")

	count, err := db.Domains.Count(user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestDomainRepo_IsAvailable(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")

	avail, err := db.Domains.IsAvailable("myapp")
	require.NoError(t, err)
	assert.True(t, avail)

	createTestDomain(t, db, user.ID, "myapp")

	avail, err = db.Domains.IsAvailable("myapp")
	require.NoError(t, err)
	assert.False(t, avail)
}

func TestDomainRepo_IsOwnedByUser(t *testing.T) {
	db := newTestDB(t)
	user1 := createTestUser(t, db, "+1111111111")
	user2 := createTestUser(t, db, "+2222222222")
	createTestDomain(t, db, user1.ID, "myapp")

	owned, err := db.Domains.IsOwnedByUser("myapp", user1.ID)
	require.NoError(t, err)
	assert.True(t, owned)

	owned, err = db.Domains.IsOwnedByUser("myapp", user2.ID)
	require.NoError(t, err)
	assert.False(t, owned)
}

func TestDomainRepo_CascadeDeleteUser(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db, "+1111111111")
	createTestDomain(t, db, user.ID, "myapp")

	require.NoError(t, db.Users.Delete(user.ID))

	domains, err := db.Domains.GetByUserID(user.ID)
	require.NoError(t, err)
	assert.Empty(t, domains)
}
