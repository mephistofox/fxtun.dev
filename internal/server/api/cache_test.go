package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The two public catalogues are read on every landing-page visit and change
// only when an admin edits them, so they are worth caching. The tag has to be
// derived from the content: that way an edit invalidates it by itself and
// nobody has to remember to bump a version.
func TestCacheableJSON_RepeatRequestIsNotModified(t *testing.T) {
	s := &Server{}
	payload := map[string]any{"plans": []string{"free", "base"}}

	first := httptest.NewRecorder()
	s.respondCacheableJSON(first, httptest.NewRequest(http.MethodGet, "/api/plans/public", nil), payload)

	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the first response")
	}
	if cc := first.Header().Get("Cache-Control"); cc == "" {
		t.Fatal("no Cache-Control on the first response")
	}
	if first.Code != http.StatusOK || first.Body.Len() == 0 {
		t.Fatalf("first response: %d, %d bytes", first.Code, first.Body.Len())
	}

	again := httptest.NewRequest(http.MethodGet, "/api/plans/public", nil)
	again.Header.Set("If-None-Match", etag)
	second := httptest.NewRecorder()
	s.respondCacheableJSON(second, again, payload)

	if second.Code != http.StatusNotModified {
		t.Fatalf("unchanged data answered %d, want 304", second.Code)
	}
	if second.Body.Len() != 0 {
		t.Fatalf("304 carried %d bytes of body", second.Body.Len())
	}
}

func TestCacheableJSON_EditedDataGetsANewTag(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/plans/public", nil)

	before := httptest.NewRecorder()
	s.respondCacheableJSON(before, req, map[string]any{"price": 210})

	after := httptest.NewRecorder()
	s.respondCacheableJSON(after, req, map[string]any{"price": 230})

	if before.Header().Get("ETag") == after.Header().Get("ETag") {
		t.Fatal("the price changed and the ETag did not: caches would keep serving the old one")
	}

	// And a client holding the old tag must be given the new body, not a 304.
	stale := httptest.NewRequest(http.MethodGet, "/api/plans/public", nil)
	stale.Header.Set("If-None-Match", before.Header().Get("ETag"))
	fresh := httptest.NewRecorder()
	s.respondCacheableJSON(fresh, stale, map[string]any{"price": 230})
	if fresh.Code != http.StatusOK {
		t.Fatalf("stale tag answered %d, want 200 with the new data", fresh.Code)
	}
}
