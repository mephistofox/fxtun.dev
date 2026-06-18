package redis

import (
	"context"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func testSessionStore(t *testing.T) *SessionStore {
	t.Helper()
	addr := os.Getenv("FXTUNNEL_TEST_REDIS")
	if addr == "" {
		t.Skip("FXTUNNEL_TEST_REDIS not set; skipping Redis-backed session store test")
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flushdb: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return NewSessionStore(&Client{rdb: rdb, prefix: "test:", log: zerolog.Nop()})
}

func TestSessionStore_RotatedTokenTracker(t *testing.T) {
	s := testSessionStore(t)

	// Unknown hash: not found.
	if _, found, err := s.RotatedOwner("never-seen"); err != nil || found {
		t.Fatalf("unknown hash: found=%v err=%v", found, err)
	}

	// Mark then look up.
	if err := s.MarkRotated("hash-a", 42, time.Minute); err != nil {
		t.Fatalf("mark: %v", err)
	}
	uid, found, err := s.RotatedOwner("hash-a")
	if err != nil || !found || uid != 42 {
		t.Fatalf("expected (42,true,nil), got (%d,%v,%v)", uid, found, err)
	}

	// Non-positive TTL is a no-op (nothing recorded).
	if err := s.MarkRotated("hash-b", 7, 0); err != nil {
		t.Fatalf("mark ttl<=0: %v", err)
	}
	if _, found, _ := s.RotatedOwner("hash-b"); found {
		t.Fatal("expected ttl<=0 mark to be a no-op")
	}
}
