package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

const (
	deviceSessionTTL      = 5 * time.Minute
	deviceCleanupInterval = 1 * time.Minute
)

const (
	deviceStatusPending    = "pending"
	deviceStatusAuthorized = "authorized"
	deviceStatusExpired    = "expired"
)

// memoryDeviceStore is the in-memory implementation of store.DeviceStore.
type memoryDeviceStore struct {
	mu       sync.RWMutex
	sessions map[string]*store.DeviceSession
}

var _ store.DeviceStore = (*memoryDeviceStore)(nil)

func newDeviceStore() *memoryDeviceStore {
	return &memoryDeviceStore{
		sessions: make(map[string]*store.DeviceSession),
	}
}

func (ds *memoryDeviceStore) Create() (*store.DeviceSession, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(bytes)

	userCode, err := generateDeviceUserCode()
	if err != nil {
		return nil, err
	}

	session := &store.DeviceSession{
		ID:        id,
		UserCode:  userCode,
		Status:    deviceStatusPending,
		CreatedAt: time.Now(),
	}

	ds.mu.Lock()
	ds.sessions[id] = session
	ds.mu.Unlock()

	return session, nil
}

// deviceUserCodeAlphabet omits characters that are easy to confuse when read
// off a terminal and typed into a browser.
const deviceUserCodeAlphabet = "BCDFGHJKMNPQRSTWXYZ23456789"

// generateDeviceUserCode returns a code in the form XXXX-XXXX.
func generateDeviceUserCode() (string, error) {
	const n = 8
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, 0, n+1)
	for i, v := range buf {
		if i == n/2 {
			out = append(out, '-')
		}
		out = append(out, deviceUserCodeAlphabet[int(v)%len(deviceUserCodeAlphabet)])
	}
	return string(out), nil
}

// GetByUserCode resolves a user-typed code to its pending session.
func (ds *memoryDeviceStore) GetByUserCode(userCode string) *store.DeviceSession {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	for id, s := range ds.sessions {
		if s.UserCode != userCode {
			continue
		}
		if time.Since(s.CreatedAt) > deviceSessionTTL {
			return &store.DeviceSession{ID: id, Status: deviceStatusExpired}
		}
		return s
	}
	return nil
}

func (ds *memoryDeviceStore) Get(id string) *store.DeviceSession {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	s, ok := ds.sessions[id]
	if !ok {
		return nil
	}
	if time.Since(s.CreatedAt) > deviceSessionTTL {
		return &store.DeviceSession{ID: id, Status: deviceStatusExpired}
	}
	return s
}

func (ds *memoryDeviceStore) Authorize(id, token string) bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	s, ok := ds.sessions[id]
	if !ok || time.Since(s.CreatedAt) > deviceSessionTTL {
		return false
	}
	s.Status = deviceStatusAuthorized
	s.Token = token
	return true
}

func (ds *memoryDeviceStore) Delete(id string) {
	ds.mu.Lock()
	delete(ds.sessions, id)
	ds.mu.Unlock()
}

func (ds *memoryDeviceStore) Cleanup(stopCh <-chan struct{}) {
	ticker := time.NewTicker(deviceCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ds.mu.Lock()
			now := time.Now()
			for id, s := range ds.sessions {
				if now.Sub(s.CreatedAt) > deviceSessionTTL*2 {
					delete(ds.sessions, id)
				}
			}
			ds.mu.Unlock()
		case <-stopCh:
			return
		}
	}
}
