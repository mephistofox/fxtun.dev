package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	deviceSessionTTL      = 5 * time.Minute
	deviceCleanupInterval = 1 * time.Minute
)

type deviceSessionStatus string

const (
	deviceStatusPending    deviceSessionStatus = "pending"
	deviceStatusAuthorized deviceSessionStatus = "authorized"
	deviceStatusExpired    deviceSessionStatus = "expired"
)

type deviceSession struct {
	ID        string
	Status    deviceSessionStatus
	Token     string
	CreatedAt time.Time
}

type deviceStore struct {
	mu       sync.RWMutex
	sessions map[string]*deviceSession
}

func newDeviceStore() *deviceStore {
	return &deviceStore{
		sessions: make(map[string]*deviceSession),
	}
}

func (ds *deviceStore) Create() (*deviceSession, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(bytes)

	session := &deviceSession{
		ID:        id,
		Status:    deviceStatusPending,
		CreatedAt: time.Now(),
	}

	ds.mu.Lock()
	ds.sessions[id] = session
	ds.mu.Unlock()

	return session, nil
}

func (ds *deviceStore) Get(id string) *deviceSession {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	s, ok := ds.sessions[id]
	if !ok {
		return nil
	}
	if time.Since(s.CreatedAt) > deviceSessionTTL {
		return &deviceSession{ID: id, Status: deviceStatusExpired}
	}
	return s
}

func (ds *deviceStore) Authorize(id, token string) bool {
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

func (ds *deviceStore) Delete(id string) {
	ds.mu.Lock()
	delete(ds.sessions, id)
	ds.mu.Unlock()
}

func (ds *deviceStore) Cleanup(stopCh <-chan struct{}) {
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
