package inspect

import (
	"net/http"
	"testing"
	"time"
)

func TestCapturedExchange_Summary(t *testing.T) {
	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	ex := &CapturedExchange{
		ID:               "ex-1",
		TunnelID:         "tun-1",
		Timestamp:        ts,
		Duration:         150 * time.Millisecond,
		Method:           "POST",
		Path:             "/api/data",
		Host:             "test.example.com",
		RequestHeaders:   http.Header{"Content-Type": {"application/json"}},
		RequestBody:      []byte(`{"key":"value"}`),
		RequestBodySize:  15,
		RemoteAddr:       "192.168.1.1:54321",
		StatusCode:       201,
		ResponseHeaders:  http.Header{"X-Custom": {"val"}},
		ResponseBody:     []byte(`{"ok":true}`),
		ResponseBodySize: 11,
	}

	s := ex.Summary()

	if s.ID != ex.ID {
		t.Errorf("ID: got %q, want %q", s.ID, ex.ID)
	}
	if s.TunnelID != ex.TunnelID {
		t.Errorf("TunnelID: got %q, want %q", s.TunnelID, ex.TunnelID)
	}
	if !s.Timestamp.Equal(ex.Timestamp) {
		t.Errorf("Timestamp: got %v, want %v", s.Timestamp, ex.Timestamp)
	}
	if s.Duration != ex.Duration {
		t.Errorf("Duration: got %v, want %v", s.Duration, ex.Duration)
	}
	if s.Method != ex.Method {
		t.Errorf("Method: got %q, want %q", s.Method, ex.Method)
	}
	if s.Path != ex.Path {
		t.Errorf("Path: got %q, want %q", s.Path, ex.Path)
	}
	if s.Host != ex.Host {
		t.Errorf("Host: got %q, want %q", s.Host, ex.Host)
	}
	if s.StatusCode != ex.StatusCode {
		t.Errorf("StatusCode: got %d, want %d", s.StatusCode, ex.StatusCode)
	}
	if s.RequestBodySize != ex.RequestBodySize {
		t.Errorf("RequestBodySize: got %d, want %d", s.RequestBodySize, ex.RequestBodySize)
	}
	if s.ResponseBodySize != ex.ResponseBodySize {
		t.Errorf("ResponseBodySize: got %d, want %d", s.ResponseBodySize, ex.ResponseBodySize)
	}
	if s.RemoteAddr != ex.RemoteAddr {
		t.Errorf("RemoteAddr: got %q, want %q", s.RemoteAddr, ex.RemoteAddr)
	}
}
