package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
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
	states map[string]*oauthStateEntry // legacy fallback; stateful usage disabled when signingKey is set
	codes  map[string]*oauthCodeEntry
	// signingKey makes state tokens stateless: the nonce is an HMAC-signed
	// serialized entry so it survives server restarts (in-memory map doesn't).
	signingKey []byte
}

func newOAuthStore() *oauthStore {
	return &oauthStore{
		states: make(map[string]*oauthStateEntry),
		codes:  make(map[string]*oauthCodeEntry),
	}
}

// SetSigningKey enables stateless OAuth state tokens. Once set, CreateState
// emits a signed self-contained nonce and ConsumeState verifies without a map.
// Server restarts will no longer invalidate in-flight OAuth flows.
func (s *oauthStore) SetSigningKey(key []byte) {
	if len(key) == 0 {
		return
	}
	derived := sha256.Sum256(append([]byte("fxtunnel-oauth-state-v1:"), key...))
	s.signingKey = derived[:]
}

func (s *oauthStore) generateNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateState returns an OAuth state nonce. When a signing key is configured
// the nonce is stateless (payload+HMAC) and survives restarts; otherwise it's
// an opaque random id stored in memory.
func (s *oauthStore) CreateState(entry *oauthStateEntry) (string, error) {
	entry.CreatedAt = time.Now()

	if len(s.signingKey) > 0 {
		return s.signStateEntry(entry)
	}

	nonce, err := s.generateNonce()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.states[nonce] = entry
	s.mu.Unlock()
	return nonce, nil
}

// ConsumeState verifies the state token. For stateless tokens (signed),
// verification is cryptographic; for legacy in-memory states the entry is
// looked up and deleted. Returns nil on any failure (invalid, expired, unknown).
func (s *oauthStore) ConsumeState(nonce string) *oauthStateEntry {
	if len(s.signingKey) > 0 && strings.Contains(nonce, ".") {
		entry, err := s.verifyStateToken(nonce)
		if err != nil {
			return nil
		}
		return entry
	}

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

// signStateEntry produces "<base64(json)>.<base64(hmac)>" — self-contained and
// tamper-evident. No server state required.
func (s *oauthStore) signStateEntry(entry *oauthStateEntry) (string, error) {
	payload, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.signingKey)
	mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return body + "." + sig, nil
}

// verifyStateToken re-derives the HMAC and parses the payload. Fails on
// bad format, bad signature, or expiry.
func (s *oauthStore) verifyStateToken(token string) (*oauthStateEntry, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("malformed state")
	}
	expected := hmac.New(sha256.New, s.signingKey)
	expected.Write([]byte(parts[0]))
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	if !hmac.Equal(expected.Sum(nil), gotSig) {
		return nil, errors.New("bad signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var entry oauthStateEntry
	if err := json.Unmarshal(payload, &entry); err != nil {
		return nil, err
	}
	if time.Since(entry.CreatedAt) > oauthStateTTL {
		return nil, errors.New("expired")
	}
	return &entry, nil
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
