package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	oauthStateTTL        = 10 * time.Minute
	oauthCodeTTL         = 2 * time.Minute
	oauthCleanupInterval = 1 * time.Minute
)

type oauthPurpose string

const (
	oauthPurposeLogin oauthPurpose = "login"
	oauthPurposeLink  oauthPurpose = "link"
)

type oauthStateEntry struct {
	Purpose         oauthPurpose
	UserID          int64
	DesktopRedirect string
	CreatedAt       time.Time
}

type oauthCodeEntry struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	CreatedAt    time.Time
}

type oauthStore struct {
	mu     sync.Mutex
	states map[string]*oauthStateEntry
	codes  map[string]*oauthCodeEntry
}

func newOAuthStore() *oauthStore {
	return &oauthStore{
		states: make(map[string]*oauthStateEntry),
		codes:  make(map[string]*oauthCodeEntry),
	}
}

func (s *oauthStore) generateNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateState stores an OAuth state entry and returns its nonce.
func (s *oauthStore) CreateState(entry *oauthStateEntry) (string, error) {
	nonce, err := s.generateNonce()
	if err != nil {
		return "", err
	}
	entry.CreatedAt = time.Now()

	s.mu.Lock()
	s.states[nonce] = entry
	s.mu.Unlock()

	return nonce, nil
}

// ConsumeState retrieves and deletes the state entry for the given nonce.
func (s *oauthStore) ConsumeState(nonce string) *oauthStateEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.states[nonce]
	if !ok {
		return nil
	}
	delete(s.states, nonce)

	if time.Since(entry.CreatedAt) > oauthStateTTL {
		return nil
	}
	return entry
}

// CreateCode stores a one-time authorization code that can be exchanged for tokens.
func (s *oauthStore) CreateCode(accessToken, refreshToken string, expiresIn int64) (string, error) {
	code, err := s.generateNonce()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.codes[code] = &oauthCodeEntry{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		CreatedAt:    time.Now(),
	}
	s.mu.Unlock()

	return code, nil
}

// ExchangeCode retrieves and deletes the code entry (one-time use).
func (s *oauthStore) ExchangeCode(code string) *oauthCodeEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.codes[code]
	if !ok {
		return nil
	}
	delete(s.codes, code)

	if time.Since(entry.CreatedAt) > oauthCodeTTL {
		return nil
	}
	return entry
}

// Cleanup removes expired entries periodically.
func (s *oauthStore) Cleanup(stopCh <-chan struct{}) {
	ticker := time.NewTicker(oauthCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, entry := range s.states {
				if now.Sub(entry.CreatedAt) > oauthStateTTL*2 {
					delete(s.states, id)
				}
			}
			for id, entry := range s.codes {
				if now.Sub(entry.CreatedAt) > oauthCodeTTL*2 {
					delete(s.codes, id)
				}
			}
			s.mu.Unlock()
		case <-stopCh:
			return
		}
	}
}
