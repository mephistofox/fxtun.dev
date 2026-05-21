package api

import (
	"testing"
	"time"
)

func TestMemIPBanStore_BanAndIsBanned(t *testing.T) {
	store := newMemIPBanStore()

	isNew, err := store.Ban("1.2.3.4", "test", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Ban returned error: %v", err)
	}
	if !isNew {
		t.Fatalf("expected isNew=true for first ban")
	}

	banned, reason, err := store.IsBanned("1.2.3.4")
	if err != nil {
		t.Fatalf("IsBanned returned error: %v", err)
	}
	if !banned {
		t.Fatalf("expected banned=true")
	}
	if reason != "test" {
		t.Fatalf("expected reason=test, got %q", reason)
	}
}

func TestMemIPBanStore_RepeatedBanIsNotNew(t *testing.T) {
	store := newMemIPBanStore()

	if _, err := store.Ban("9.9.9.9", "first", time.Second); err != nil {
		t.Fatalf("first Ban error: %v", err)
	}
	isNew, err := store.Ban("9.9.9.9", "second", time.Second)
	if err != nil {
		t.Fatalf("second Ban error: %v", err)
	}
	if isNew {
		t.Fatalf("expected isNew=false on repeated ban of same active IP")
	}
}

func TestMemIPBanStore_Expires(t *testing.T) {
	store := newMemIPBanStore()

	if _, err := store.Ban("5.5.5.5", "short", 10*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)

	banned, _, err := store.IsBanned("5.5.5.5")
	if err != nil {
		t.Fatal(err)
	}
	if banned {
		t.Fatalf("expected ban to expire after TTL")
	}
}

func TestMemIPBanStore_Unban(t *testing.T) {
	store := newMemIPBanStore()

	if _, err := store.Ban("8.8.8.8", "abuse", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.Unban("8.8.8.8"); err != nil {
		t.Fatal(err)
	}
	banned, _, _ := store.IsBanned("8.8.8.8")
	if banned {
		t.Fatalf("expected unban to clear ban")
	}
}

func TestMemIPBanStore_List(t *testing.T) {
	store := newMemIPBanStore()

	_, _ = store.Ban("1.1.1.1", "a", time.Hour)
	_, _ = store.Ban("2.2.2.2", "b", time.Hour)
	_, _ = store.Ban("3.3.3.3", "c", 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)

	entries, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 active bans, got %d", len(entries))
	}
}

func TestMemIPBanStore_EmptyIPNoop(t *testing.T) {
	store := newMemIPBanStore()
	if isNew, err := store.Ban("", "x", time.Hour); err != nil || isNew {
		t.Fatalf("empty IP should be a no-op, got isNew=%v err=%v", isNew, err)
	}
	if banned, _, _ := store.IsBanned(""); banned {
		t.Fatalf("empty IP should never be banned")
	}
}
