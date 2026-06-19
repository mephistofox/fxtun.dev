package auth

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/server/database"
	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

// fakeSessionStore is an in-memory SessionStore that also implements
// store.RotatedTokenTracker, so it exercises refresh-token reuse detection
// without a real Redis.
type fakeSessionStore struct {
	mu      sync.Mutex
	byHash  map[string]*database.Session
	rotated map[string]int64
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{byHash: map[string]*database.Session{}, rotated: map[string]int64{}}
}

var (
	_ store.SessionStore        = (*fakeSessionStore)(nil)
	_ store.RotatedTokenTracker = (*fakeSessionStore)(nil)
)

func (f *fakeSessionStore) Create(s *database.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *s
	f.byHash[s.RefreshTokenHash] = &cp
	return nil
}

func (f *fakeSessionStore) GetByTokenHash(h string) (*database.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.byHash[h]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, database.ErrSessionNotFound
}

func (f *fakeSessionStore) GetByUserID(userID int64) ([]*database.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*database.Session
	for _, s := range f.byHash {
		if s.UserID == userID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeSessionStore) Delete(int64) error { return nil }

func (f *fakeSessionStore) DeleteByTokenHash(h string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byHash, h)
	return nil
}

func (f *fakeSessionStore) DeleteByUserID(userID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for h, s := range f.byHash {
		if s.UserID == userID {
			delete(f.byHash, h)
		}
	}
	return nil
}

func (f *fakeSessionStore) DeleteExpired() (int64, error) { return 0, nil }

func (f *fakeSessionStore) MarkRotated(h string, userID int64, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rotated[h] = userID
	return nil
}

func (f *fakeSessionStore) RotatedOwner(h string) (int64, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	uid, ok := f.rotated[h]
	return uid, ok, nil
}

func setupAuthTestUser(t *testing.T) (*database.Database, *database.User) {
	t.Helper()
	dsn := os.Getenv("FXTUNNEL_TEST_DSN")
	if dsn == "" {
		t.Skip("FXTUNNEL_TEST_DSN not set; skipping Postgres-backed auth test")
	}
	log := zerolog.New(zerolog.NewTestWriter(t))
	db, err := database.New(dsn, log)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if _, err := db.Pool().Exec(context.Background(),
		"TRUNCATE users, sessions RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	free, err := db.Plans.GetBySlug("free")
	if err != nil {
		t.Fatalf("free plan: %v", err)
	}
	user := &database.User{Phone: "+79995550001", PasswordHash: "hash", PlanID: free.ID, IsActive: true}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return db, user
}

// Reusing an already-rotated refresh token must be detected as theft and revoke
// the user's entire session family.
func TestRefreshTokens_ReuseRevokesFamily(t *testing.T) {
	db, user := setupAuthTestUser(t)
	log := zerolog.New(zerolog.NewTestWriter(t))

	svc := NewService(db, "test-secret", time.Hour, 24*time.Hour, "fxtunnel", make([]byte, 32), 1, log)
	fake := newFakeSessionStore()
	svc.SetSessionStore(fake)

	// Seed an initial session for an opaque refresh token.
	const firstToken = "refresh-token-one"
	if err := fake.Create(&database.Session{
		UserID:           user.ID,
		RefreshTokenHash: HashToken(firstToken),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		CreatedAt:        time.Now(),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	// First refresh rotates the token successfully.
	_, pair, err := svc.RefreshTokens(firstToken, "ua", "1.2.3.4")
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if pair == nil || pair.RefreshToken == "" {
		t.Fatal("expected a new token pair")
	}

	// Reusing the now-rotated first token must be detected and revoke the family.
	_, _, err = svc.RefreshTokens(firstToken, "ua", "1.2.3.4")
	if !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected ErrTokenReuse on reuse, got %v", err)
	}

	sessions, err := fake.GetByUserID(user.ID)
	if err != nil {
		t.Fatalf("get sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected all sessions revoked after reuse, got %d", len(sessions))
	}
}
