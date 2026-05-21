package api

import (
	"sync"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

var _ store.IPBanStore = (*memIPBanStore)(nil)

type memIPBanEntry struct {
	reason    string
	bannedAt  time.Time
	expiresAt time.Time
}

// memIPBanStore is an in-memory IPBanStore used when Redis is unavailable.
// Entries expire lazily on read and via a janitor goroutine.
type memIPBanStore struct {
	mu      sync.RWMutex
	entries map[string]memIPBanEntry
}

func newMemIPBanStore() *memIPBanStore {
	return &memIPBanStore{entries: make(map[string]memIPBanEntry)}
}

func (m *memIPBanStore) Ban(ip, reason string, ttl time.Duration) (bool, error) {
	if ip == "" || ttl <= 0 {
		return false, nil
	}
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()

	prev, existed := m.entries[ip]
	bannedAt := now
	if existed && prev.expiresAt.After(now) && !prev.bannedAt.IsZero() {
		bannedAt = prev.bannedAt
	}
	m.entries[ip] = memIPBanEntry{
		reason:    reason,
		bannedAt:  bannedAt,
		expiresAt: now.Add(ttl),
	}
	return !existed || !prev.expiresAt.After(now), nil
}

func (m *memIPBanStore) IsBanned(ip string) (bool, string, error) {
	if ip == "" {
		return false, "", nil
	}
	now := time.Now().UTC()
	m.mu.RLock()
	entry, ok := m.entries[ip]
	m.mu.RUnlock()
	if !ok || !entry.expiresAt.After(now) {
		return false, "", nil
	}
	return true, entry.reason, nil
}

func (m *memIPBanStore) Unban(ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, ip)
	return nil
}

func (m *memIPBanStore) List() ([]store.IPBanEntry, error) {
	now := time.Now().UTC()
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]store.IPBanEntry, 0, len(m.entries))
	for ip, e := range m.entries {
		if !e.expiresAt.After(now) {
			continue
		}
		out = append(out, store.IPBanEntry{
			IP:        ip,
			Reason:    e.reason,
			BannedAt:  e.bannedAt,
			ExpiresAt: e.expiresAt,
		})
	}
	return out, nil
}

// cleanup removes expired entries periodically. Stops when stopCh is closed.
func (m *memIPBanStore) cleanup(stopCh <-chan struct{}) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-t.C:
			now := time.Now().UTC()
			m.mu.Lock()
			for ip, e := range m.entries {
				if !e.expiresAt.After(now) {
					delete(m.entries, ip)
				}
			}
			m.mu.Unlock()
		}
	}
}
