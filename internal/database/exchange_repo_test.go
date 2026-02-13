package database

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mephistofox/fxtun.dev/internal/inspect"
)

func setupExchangeTestDB(t *testing.T) (*ExchangeRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE inspect_exchanges (
			id TEXT PRIMARY KEY,
			tunnel_id TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			trace_id TEXT,
			replay_ref TEXT,
			timestamp DATETIME NOT NULL,
			duration_ns INTEGER NOT NULL,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			host TEXT NOT NULL,
			request_headers TEXT,
			request_body BLOB,
			request_body_size INTEGER NOT NULL DEFAULT 0,
			response_headers TEXT,
			response_body BLOB,
			response_body_size INTEGER NOT NULL DEFAULT 0,
			status_code INTEGER NOT NULL,
			remote_addr TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_inspect_exch_tunnel ON inspect_exchanges(tunnel_id, timestamp DESC);
		CREATE INDEX idx_inspect_exch_created ON inspect_exchanges(created_at);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return NewExchangeRepository(db), db
}

func makeTestExchange(id, tunnelID string) *inspect.CapturedExchange {
	return &inspect.CapturedExchange{
		ID:               id,
		TunnelID:         tunnelID,
		Timestamp:        time.Now(),
		Duration:         100 * time.Millisecond,
		Method:           "GET",
		Path:             "/test",
		Host:             "test.example.com",
		RequestHeaders:   http.Header{"Content-Type": {"application/json"}},
		RequestBody:      []byte(`{"key":"value"}`),
		RequestBodySize:  15,
		RemoteAddr:       "127.0.0.1",
		StatusCode:       200,
		ResponseHeaders:  http.Header{"Content-Type": {"text/html"}},
		ResponseBody:     []byte("<html></html>"),
		ResponseBodySize: 13,
	}
}

func TestExchangeRepository_SaveAndGetByID(t *testing.T) {
	repo, db := setupExchangeTestDB(t)
	defer db.Close()

	ex := makeTestExchange("ex-1", "tun-1")
	ex.TraceID = "trace-abc"
	ex.ReplayRef = "orig-123"

	if err := repo.Save(ex, 42); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := repo.GetByID("ex-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}

	if got.ID != "ex-1" {
		t.Errorf("ID = %q, want %q", got.ID, "ex-1")
	}
	if got.TunnelID != "tun-1" {
		t.Errorf("TunnelID = %q, want %q", got.TunnelID, "tun-1")
	}
	if got.TraceID != "trace-abc" {
		t.Errorf("TraceID = %q, want %q", got.TraceID, "trace-abc")
	}
	if got.ReplayRef != "orig-123" {
		t.Errorf("ReplayRef = %q, want %q", got.ReplayRef, "orig-123")
	}
	if got.Method != "GET" {
		t.Errorf("Method = %q, want %q", got.Method, "GET")
	}
	if got.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want %d", got.StatusCode, 200)
	}
	if got.RequestHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("RequestHeaders Content-Type = %q", got.RequestHeaders.Get("Content-Type"))
	}
	if string(got.RequestBody) != `{"key":"value"}` {
		t.Errorf("RequestBody = %q", string(got.RequestBody))
	}
	if string(got.ResponseBody) != "<html></html>" {
		t.Errorf("ResponseBody = %q", string(got.ResponseBody))
	}
}

func TestExchangeRepository_GetByID_NotFound(t *testing.T) {
	repo, db := setupExchangeTestDB(t)
	defer db.Close()

	got, err := repo.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent ID")
	}
}

func TestExchangeRepository_ListByTunnelID(t *testing.T) {
	repo, db := setupExchangeTestDB(t)
	defer db.Close()

	// Insert 5 exchanges for tunnel "tun-1"
	for i := 0; i < 5; i++ {
		ex := makeTestExchange("ex-"+string(rune('a'+i)), "tun-1")
		ex.Timestamp = time.Now().Add(time.Duration(i) * time.Second)
		if err := repo.Save(ex, 1); err != nil {
			t.Fatalf("Save %d failed: %v", i, err)
		}
	}
	// Insert 1 exchange for different tunnel
	ex := makeTestExchange("ex-other", "tun-2")
	if err := repo.Save(ex, 1); err != nil {
		t.Fatalf("Save other failed: %v", err)
	}

	// List all for tun-1
	exchanges, total, err := repo.ListByTunnelID("tun-1", 0, 10)
	if err != nil {
		t.Fatalf("ListByTunnelID failed: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(exchanges) != 5 {
		t.Errorf("len(exchanges) = %d, want 5", len(exchanges))
	}

	// Verify newest-first ordering
	if len(exchanges) >= 2 && exchanges[0].Timestamp.Before(exchanges[1].Timestamp) {
		t.Error("exchanges not in newest-first order")
	}

	// Test pagination
	exchanges, total, err = repo.ListByTunnelID("tun-1", 2, 2)
	if err != nil {
		t.Fatalf("ListByTunnelID paginated failed: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(exchanges) != 2 {
		t.Errorf("len(exchanges) = %d, want 2", len(exchanges))
	}
}

func TestExchangeRepository_DeleteOlderThan(t *testing.T) {
	repo, db := setupExchangeTestDB(t)
	defer db.Close()

	// Insert exchange with created_at in the past
	ex := makeTestExchange("ex-old", "tun-1")
	if err := repo.Save(ex, 1); err != nil {
		t.Fatal(err)
	}
	// Manually set created_at to 48h ago
	_, err := db.Exec("UPDATE inspect_exchanges SET created_at = ? WHERE id = ?",
		time.Now().Add(-48*time.Hour), "ex-old")
	if err != nil {
		t.Fatal(err)
	}

	// Insert recent exchange
	ex2 := makeTestExchange("ex-new", "tun-1")
	if err := repo.Save(ex2, 1); err != nil {
		t.Fatal(err)
	}

	deleted, err := repo.DeleteOlderThan(time.Now().Add(-24 * time.Hour))
	if err != nil {
		t.Fatalf("DeleteOlderThan failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify old is gone, new remains
	got, _ := repo.GetByID("ex-old")
	if got != nil {
		t.Error("old exchange should be deleted")
	}
	got, _ = repo.GetByID("ex-new")
	if got == nil {
		t.Error("new exchange should still exist")
	}
}

func TestExchangeRepository_DeleteByTunnelID(t *testing.T) {
	repo, db := setupExchangeTestDB(t)
	defer db.Close()

	ex1 := makeTestExchange("ex-1", "tun-1")
	ex2 := makeTestExchange("ex-2", "tun-1")
	ex3 := makeTestExchange("ex-3", "tun-2")
	for _, ex := range []*inspect.CapturedExchange{ex1, ex2, ex3} {
		if err := repo.Save(ex, 1); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := repo.DeleteByTunnelID("tun-1")
	if err != nil {
		t.Fatalf("DeleteByTunnelID failed: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}

	// tun-2 exchange should remain
	got, _ := repo.GetByID("ex-3")
	if got == nil {
		t.Error("tun-2 exchange should still exist")
	}
}
