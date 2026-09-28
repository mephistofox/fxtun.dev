package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Reporting "ok" while the database is unreachable tells whoever watches this
// endpoint the opposite of the truth: every request that touches storage is
// failing at that moment.
func TestHealth_ReportsDatabaseState(t *testing.T) {
	env := setupTestEnv(t)

	resp, err := http.Get(env.Server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 with a live database", resp.StatusCode)
	}

	var body struct {
		Status   string `json:"status"`
		Database string `json:"database"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Database != "" {
		t.Errorf("database = %q, want it omitted while healthy", body.Database)
	}

	// With the pool closed the endpoint must stop claiming everything is fine.
	env.DB.Pool().Close()

	resp2, err := http.Get(env.Server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health after closing the pool: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d with an unusable database, want 503", resp2.StatusCode)
	}
}
