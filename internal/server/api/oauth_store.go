package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

const (
	oauthStateTTL        = 10 * time.Minute
	oauthCodeTTL         = 2 * time.Minute
	oauthCleanupInterval = 1 * time.Minute
)

const (
	oauthPurposeLogin = "login"
	oauthPurposeLink  = "link"
)

type oauthStateInternal struct {
	entry     *store.OAuthStateEntry
	createdAt time.Time
}

type oauthCodeInternal struct {
	entry     *store.OAuthCodeEntry
	createdAt time.Time
}

// memoryOAuthStore is the in-memory implementation of store.OAuthStore.
type memoryOAuthStore struct {
	mu     sync.Mutex
	states map[string]*oauthStateInternal
	codes  map[string]*oauthCodeInternal
}

var _ store.OAuthStore = (*memoryOAuthStore)(nil)

func newOAuthStore() *memoryOAuthStore {
	return &memoryOAuthStore{
		states: make(map[string]*oauthStateInternal),
		codes:  make(map[string]*oauthCodeInternal),
	}
}

func (s *memoryOAuthStore) generateNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateState stores an OAuth state entry and returns its nonce.
func (s *memoryOAuthStore) CreateState(entry *store.OAuthStateEntry) (string, error) {
	nonce, err := s.generateNonce()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.states[nonce] = &oauthStateInternal{entry: entry, createdAt: time.Now()}
	s.mu.Unlock()

	return nonce, nil
}

// ConsumeState retrieves and deletes the state entry for the given nonce.
func (s *memoryOAuthStore) ConsumeState(nonce string) *store.OAuthStateEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	internal, ok := s.states[nonce]
	if !ok {
		return nil
	}
	delete(s.states, nonce)

	if time.Since(internal.createdAt) > oauthStateTTL {
		return nil
	}
	return internal.entry
}

// CreateCode stores a one-time authorization code that can be exchanged for tokens.
func (s *memoryOAuthStore) CreateCode(accessToken, refreshToken string, expiresIn int64) (string, error) {
	code, err := s.generateNonce()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.codes[code] = &oauthCodeInternal{
		entry: &store.OAuthCodeEntry{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    expiresIn,
		},
		createdAt: time.Now(),
	}
	s.mu.Unlock()

	return code, nil
}

// ExchangeCode retrieves and deletes the code entry (one-time use).
func (s *memoryOAuthStore) ExchangeCode(code string) *store.OAuthCodeEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	internal, ok := s.codes[code]
	if !ok {
		return nil
	}
	delete(s.codes, code)

	if time.Since(internal.createdAt) > oauthCodeTTL {
		return nil
	}
	return internal.entry
}

// Cleanup removes expired entries periodically.
func (s *memoryOAuthStore) Cleanup(stopCh <-chan struct{}) {
	ticker := time.NewTicker(oauthCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, internal := range s.states {
				if now.Sub(internal.createdAt) > oauthStateTTL*2 {
					delete(s.states, id)
				}
			}
			for id, internal := range s.codes {
				if now.Sub(internal.createdAt) > oauthCodeTTL*2 {
					delete(s.codes, id)
				}
			}
			s.mu.Unlock()
		case <-stopCh:
			return
		}
	}
}
