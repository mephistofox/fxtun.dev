package redis

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/server/store"
)

// testRegistry connects to the Redis given by FXTUNNEL_TEST_REDIS, flushes it,
// and returns a registry. Skips when no Redis is configured.
func testRegistry(t *testing.T) (*TunnelRegistry, *Client) {
	t.Helper()
	addr := os.Getenv("FXTUNNEL_TEST_REDIS")
	if addr == "" {
		t.Skip("FXTUNNEL_TEST_REDIS not set; skipping Redis-backed registry test")
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping: %v", err)
	}
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flushdb: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	c := &Client{rdb: rdb, prefix: "test:", log: zerolog.Nop()}
	return NewTunnelRegistry(c, "server-1"), c
}

func entryFor(userID int64, tunnelID, sub string) store.TunnelEntry {
	return store.TunnelEntry{
		TunnelID:  tunnelID,
		Type:      "http",
		Subdomain: sub,
		UserID:    userID,
		ClientID:  "client-" + tunnelID,
		CreatedAt: time.Now(),
	}
}

// A tunnel must not be able to claim a subdomain already owned by a different
// user — the core of the cross-node subdomain-hijack fix.
func TestTunnelRegistry_SubdomainClaimIsOwnershipGuarded(t *testing.T) {
	reg, _ := testRegistry(t)

	if err := reg.Register(entryFor(1, "tunA", "app")); err != nil {
		t.Fatalf("A register: %v", err)
	}

	err := reg.Register(entryFor(2, "tunB", "app"))
	if !errors.Is(err, store.ErrSubdomainTaken) {
		t.Fatalf("B register: expected ErrSubdomainTaken, got %v", err)
	}

	got, err := reg.LookupBySubdomain("app")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got == nil || got.UserID != 1 || got.TunnelID != "tunA" {
		t.Fatalf("expected A's entry to remain, got %+v", got)
	}

	// The owner re-registering (heartbeat/reconnect) must still succeed.
	if err := reg.Register(entryFor(1, "tunA", "app")); err != nil {
		t.Fatalf("A re-register: %v", err)
	}
}

// Closing one tunnel must not drop a subdomain another tunnel has reclaimed.
func TestTunnelRegistry_UnregisterDoesNotDropAnotherTunnelsClaim(t *testing.T) {
	reg, c := testRegistry(t)

	if err := reg.Register(entryFor(1, "tunA", "app")); err != nil {
		t.Fatalf("A register: %v", err)
	}

	// Simulate A's sub key expiring, then user C legitimately reclaiming "app".
	c.RDB().Del(context.Background(), c.Key("tunnel", "sub", "app"))
	if err := reg.Register(entryFor(3, "tunC", "app")); err != nil {
		t.Fatalf("C register after expiry: %v", err)
	}

	// A's tunnel closes; its info still references "app", but the stale
	// Unregister must not delete C's claim.
	if err := reg.Unregister("tunA"); err != nil {
		t.Fatalf("A unregister: %v", err)
	}

	got, err := reg.LookupBySubdomain("app")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got == nil || got.TunnelID != "tunC" {
		t.Fatalf("expected C to still own 'app', got %+v", got)
	}
}
