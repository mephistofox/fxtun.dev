package api

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

const magicLinkCleanupInterval = 1 * time.Minute

type magicLinkInternal struct {
	entry     *store.MagicLinkEntry
	expiresAt time.Time
}

type magicLinkAttempt struct {
	count     int
	expiresAt time.Time
}

// memoryMagicLinkStore is the in-memory implementation of store.MagicLinkStore.
// It is used when Redis is not configured (single-node deployments).
type memoryMagicLinkStore struct {
	mu          sync.Mutex
	byToken     map[string]*magicLinkInternal
	emailToken  map[string]string            // email -> active token
	attempts    map[string]*magicLinkAttempt // email -> failed code attempts
	cooldownTil map[string]time.Time         // email -> send cooldown expiry
}

var _ store.MagicLinkStore = (*memoryMagicLinkStore)(nil)

func newMagicLinkStore() *memoryMagicLinkStore {
	return &memoryMagicLinkStore{
		byToken:     make(map[string]*magicLinkInternal),
		emailToken:  make(map[string]string),
		attempts:    make(map[string]*magicLinkAttempt),
		cooldownTil: make(map[string]time.Time),
	}
}

func (s *memoryMagicLinkStore) generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create stores a verification entry and returns its opaque token.
func (s *memoryMagicLinkStore) Create(entry *store.MagicLinkEntry, ttl time.Duration) (string, error) {
	token, err := s.generateToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Drop any previous pending token for this email.
	if old, ok := s.emailToken[entry.Email]; ok {
		delete(s.byToken, old)
	}
	s.byToken[token] = &magicLinkInternal{entry: entry, expiresAt: time.Now().Add(ttl)}
	s.emailToken[entry.Email] = token
	delete(s.attempts, entry.Email)
	return token, nil
}

// ConsumeToken retrieves and deletes the entry for a link token (one-time use).
func (s *memoryMagicLinkStore) ConsumeToken(token string) *store.MagicLinkEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	internal, ok := s.byToken[token]
	if !ok {
		return nil
	}
	delete(s.byToken, token)
	delete(s.emailToken, internal.entry.Email)
	delete(s.attempts, internal.entry.Email)

	if time.Now().After(internal.expiresAt) {
		return nil
	}
	return internal.entry
}

// LookupByToken returns the entry for a link token without consuming it.
func (s *memoryMagicLinkStore) LookupByToken(token string) (*store.MagicLinkEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	internal, ok := s.byToken[token]
	if !ok || time.Now().After(internal.expiresAt) {
		return nil, false
	}
	return internal.entry, true
}

// LookupByEmail returns the active token/entry for an email without consuming it.
func (s *memoryMagicLinkStore) LookupByEmail(email string) (string, *store.MagicLinkEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.emailToken[email]
	if !ok {
		return "", nil, false
	}
	internal, ok := s.byToken[token]
	if !ok || time.Now().After(internal.expiresAt) {
		return "", nil, false
	}
	return token, internal.entry, true
}

// IncrAttempt increments and returns the failed-code attempt counter for email.
func (s *memoryMagicLinkStore) IncrAttempt(email string, ttl time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.attempts[email]
	if !ok || time.Now().After(a.expiresAt) {
		a = &magicLinkAttempt{}
		s.attempts[email] = a
	}
	a.count++
	a.expiresAt = time.Now().Add(ttl)
	return a.count
}

// AllowSend reports whether a new link may be sent now, recording a cooldown.
func (s *memoryMagicLinkStore) AllowSend(email string, cooldown time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if til, ok := s.cooldownTil[email]; ok && now.Before(til) {
		return false
	}
	s.cooldownTil[email] = now.Add(cooldown)
	return true
}

// Cleanup removes expired entries periodically.
func (s *memoryMagicLinkStore) Cleanup(stopCh <-chan struct{}) {
	ticker := time.NewTicker(magicLinkCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for token, internal := range s.byToken {
				if now.After(internal.expiresAt) {
					delete(s.emailToken, internal.entry.Email)
					delete(s.byToken, token)
				}
			}
			for email, a := range s.attempts {
				if now.After(a.expiresAt) {
					delete(s.attempts, email)
				}
			}
			for email, til := range s.cooldownTil {
				if now.After(til) {
					delete(s.cooldownTil, email)
				}
			}
			s.mu.Unlock()
		case <-stopCh:
			return
		}
	}
}
