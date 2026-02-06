package inspect

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_CreateAndGet(t *testing.T) {
	m := NewManager(64, 4096)
	require.True(t, m.Enabled())

	buf1 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf1)

	buf2 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf2)

	assert.Same(t, buf1, buf2, "GetOrCreate should return the same pointer")

	buf3 := m.Get("tunnel-1")
	assert.Same(t, buf1, buf3, "Get should return the same pointer")

	assert.Nil(t, m.Get("nonexistent"))
}

func TestManager_Remove(t *testing.T) {
	m := NewManager(64, 4096)

	buf1 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf1)

	m.Remove("tunnel-1")
	assert.Nil(t, m.Get("tunnel-1"))

	buf2 := m.GetOrCreate("tunnel-1")
	require.NotNil(t, buf2)
	assert.NotSame(t, buf1, buf2, "after Remove, GetOrCreate should return a fresh buffer")
	assert.Equal(t, 0, buf2.Len())
}

func TestManager_Disabled(t *testing.T) {
	m := NewManager(0, 4096)
	assert.False(t, m.Enabled())
	assert.Nil(t, m.GetOrCreate("tunnel-1"))
}

func TestManager_GetOrCreateWithUser(t *testing.T) {
	m := NewManager(64, 4096)

	buf := m.GetOrCreateWithUser("tunnel-1", 42)
	require.NotNil(t, buf)

	// Verify same buffer returned
	buf2 := m.GetOrCreateWithUser("tunnel-1", 42)
	assert.Same(t, buf, buf2)
}

// mockStore is a thread-safe in-memory store for testing.
type mockStore struct {
	mu    sync.Mutex
	saved []*CapturedExchange
}

func (s *mockStore) Save(ex *CapturedExchange, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = append(s.saved, ex)
	return nil
}

func (s *mockStore) ListByTunnelID(tunnelID string, offset, limit int) ([]*CapturedExchange, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*CapturedExchange
	for _, ex := range s.saved {
		if ex.TunnelID == tunnelID {
			result = append(result, ex)
		}
	}
	total := len(result)
	if offset > len(result) {
		return nil, total, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, total, nil
}

func (s *mockStore) ListByHostAndUser(host string, userID int64, offset, limit int) ([]*CapturedExchange, int, error) {
	return nil, 0, nil
}

func (s *mockStore) GetByID(id string) (*CapturedExchange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ex := range s.saved {
		if ex.ID == id {
			return ex, nil
		}
	}
	return nil, nil
}

func (s *mockStore) DeleteByTunnelID(tunnelID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var kept []*CapturedExchange
	var deleted int64
	for _, ex := range s.saved {
		if ex.TunnelID == tunnelID {
			deleted++
		} else {
			kept = append(kept, ex)
		}
	}
	s.saved = kept
	return deleted, nil
}

func (s *mockStore) getSaved() []*CapturedExchange {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]*CapturedExchange, len(s.saved))
	copy(cp, s.saved)
	return cp
}

func TestManager_AddAndPersist(t *testing.T) {
	m := NewManager(64, 4096)
	store := &mockStore{}
	m.SetStore(store)

	m.GetOrCreateWithUser("tunnel-1", 42)

	ex := &CapturedExchange{ID: "ex-1", TunnelID: "tunnel-1", Method: "GET", Path: "/test"}
	m.AddAndPersist("tunnel-1", ex)

	// Persist is synchronous, data should be available immediately
	saved := store.getSaved()
	assert.Len(t, saved, 1)
	assert.Equal(t, "ex-1", saved[0].ID)

	m.Close()
}

func TestManager_AddAndPersist_NoUserID(t *testing.T) {
	m := NewManager(64, 4096)
	store := &mockStore{}
	m.SetStore(store)

	m.GetOrCreate("tunnel-1") // no user ID

	ex := &CapturedExchange{ID: "ex-1", TunnelID: "tunnel-1"}
	m.AddAndPersist("tunnel-1", ex)

	assert.Len(t, store.getSaved(), 0, "should not persist without user ID")
}

func TestManager_ListPersisted(t *testing.T) {
	m := NewManager(64, 4096)
	store := &mockStore{
		saved: []*CapturedExchange{
			{ID: "ex-1", TunnelID: "tun-1"},
			{ID: "ex-2", TunnelID: "tun-1"},
			{ID: "ex-3", TunnelID: "tun-2"},
		},
	}
	m.SetStore(store)

	result, total, err := m.ListPersisted("tun-1", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, result, 2)

	// No store
	m2 := NewManager(64, 4096)
	result2, total2, err2 := m2.ListPersisted("tun-1", 0, 10)
	require.NoError(t, err2)
	assert.Equal(t, 0, total2)
	assert.Nil(t, result2)
}

func TestManager_GetPersisted(t *testing.T) {
	m := NewManager(64, 4096)
	store := &mockStore{
		saved: []*CapturedExchange{
			{ID: "ex-1", TunnelID: "tun-1"},
		},
	}
	m.SetStore(store)

	ex, err := m.GetPersisted("ex-1")
	require.NoError(t, err)
	require.NotNil(t, ex)
	assert.Equal(t, "ex-1", ex.ID)

	ex2, err := m.GetPersisted("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, ex2)
}
