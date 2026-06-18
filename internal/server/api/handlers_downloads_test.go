package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

func TestParseBinaryName_SkipsSignatures(t *testing.T) {
	if _, _, ok := parseBinaryName("fxtunnel-linux-amd64.sig"); ok {
		t.Fatal("a .sig file must not be parsed as a downloadable platform")
	}
	if _, _, ok := parseBinaryName("fxtunnel-gui-windows-amd64.exe.sig"); ok {
		t.Fatal("a .sig file must not be parsed as a downloadable platform")
	}
	// Sanity: a real binary still parses.
	if _, _, ok := parseBinaryName("fxtunnel-linux-amd64"); !ok {
		t.Fatal("a real binary must still parse")
	}
}

func downloadReq(t *testing.T, platform string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/downloads/"+platform, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("platform", platform)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestHandleDownload_ServesBinaryAndSignature(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fxtunnel-linux-amd64"), []byte("BINARY"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fxtunnel-linux-amd64.sig"), []byte("deadbeef"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{downloadsPath: dir, log: zerolog.Nop()}

	t.Run("binary", func(t *testing.T) {
		rec := httptest.NewRecorder()
		s.handleDownload(rec, downloadReq(t, "cli-linux-amd64"))
		if rec.Code != http.StatusOK || rec.Body.String() != "BINARY" {
			t.Fatalf("binary: code=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("signature", func(t *testing.T) {
		rec := httptest.NewRecorder()
		s.handleDownload(rec, downloadReq(t, "cli-linux-amd64.sig"))
		if rec.Code != http.StatusOK || rec.Body.String() != "deadbeef" {
			t.Fatalf("signature: code=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing signature is 404", func(t *testing.T) {
		_ = os.Remove(filepath.Join(dir, "fxtunnel-linux-amd64.sig"))
		rec := httptest.NewRecorder()
		s.handleDownload(rec, downloadReq(t, "cli-linux-amd64.sig"))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("missing sig: expected 404, got %d", rec.Code)
		}
	})
}
