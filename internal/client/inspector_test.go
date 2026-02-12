package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mephistofox/fxtunnel/internal/inspect"
)

func newTestInspector() *Inspector {
	mgr := inspect.NewManager(1000, 262144)
	return NewInspector(mgr, "127.0.0.1:0", 262144, zerolog.Nop())
}

func addTestExchange(mgr *inspect.Manager, tunnelID, method, pathStr string, status int) *inspect.CapturedExchange {
	buf := mgr.GetOrCreate(tunnelID)
	ex := &inspect.CapturedExchange{
		ID:               generateID(),
		TunnelID:         tunnelID,
		Timestamp:        time.Now(),
		Duration:         50 * time.Millisecond,
		Method:           method,
		Path:             pathStr,
		Host:             "test.fxtun.dev",
		StatusCode:       status,
		RequestHeaders:   http.Header{"Content-Type": {"application/json"}},
		RequestBody:      []byte(`{"test":true}`),
		RequestBodySize:  13,
		ResponseHeaders:  http.Header{"Content-Type": {"application/json"}},
		ResponseBody:     []byte(`{"ok":true}`),
		ResponseBodySize: 11,
		RemoteAddr:       "192.168.1.1:54321",
	}
	buf.Add(ex)
	return ex
}

func TestInspectorListExchanges(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/api/v1/users", 200)
	addTestExchange(insp.manager, "tun-1", "POST", "/api/v1/users", 201)

	req := httptest.NewRequest("GET", "/api/requests/http", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Requests []json.RawMessage `json:"requests"`
		Total    int               `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total)
	assert.Len(t, resp.Requests, 2)
}

func TestInspectorGetExchange(t *testing.T) {
	insp := newTestInspector()
	ex := addTestExchange(insp.manager, "tun-1", "GET", "/api/v1/health", 200)

	req := httptest.NewRequest("GET", "/api/requests/http/"+ex.ID, nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got inspect.CapturedExchange
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, ex.ID, got.ID)
	assert.Equal(t, "GET", got.Method)
	assert.Equal(t, "/api/v1/health", got.Path)
	assert.Equal(t, 200, got.StatusCode)
	assert.Equal(t, "test.fxtun.dev", got.Host)
}

func TestInspectorGetExchangeNotFound(t *testing.T) {
	insp := newTestInspector()

	req := httptest.NewRequest("GET", "/api/requests/http/nonexistent-id", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestInspectorDeleteExchanges(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/a", 200)
	addTestExchange(insp.manager, "tun-1", "POST", "/b", 201)

	req := httptest.NewRequest("DELETE", "/api/requests/http", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify cleared.
	listReq := httptest.NewRequest("GET", "/api/requests/http", nil)
	listRec := httptest.NewRecorder()
	insp.ServeHTTP(listRec, listReq)

	var resp struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Total)
}

func TestInspectorSummary(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/ok1", 200)
	addTestExchange(insp.manager, "tun-1", "GET", "/ok2", 200)
	addTestExchange(insp.manager, "tun-1", "POST", "/ok3", 201)
	addTestExchange(insp.manager, "tun-1", "GET", "/err", 500)
	addTestExchange(insp.manager, "tun-1", "GET", "/notfound", 404)

	req := httptest.NewRequest("GET", "/api/requests/http/summary", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp summaryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 5, resp.Total)
	assert.Equal(t, 3, resp.ByStatus["2xx"])
	assert.Equal(t, 1, resp.ByStatus["4xx"])
	assert.Equal(t, 1, resp.ByStatus["5xx"])
	assert.Equal(t, 4, resp.ByMethod["GET"])
	assert.Equal(t, 1, resp.ByMethod["POST"])
	assert.InDelta(t, 0.4, resp.ErrorRate, 0.01)
	assert.NotNil(t, resp.LastRequestAt)
}

func TestInspectorFilterByMethod(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/a", 200)
	addTestExchange(insp.manager, "tun-1", "POST", "/b", 200)
	addTestExchange(insp.manager, "tun-1", "GET", "/c", 200)

	req := httptest.NewRequest("GET", "/api/requests/http?method=POST", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Requests []exchangeListItem `json:"requests"`
		Total    int                `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Total)
	assert.Len(t, resp.Requests, 1)
	assert.Equal(t, "POST", resp.Requests[0].Method)
}

func TestInspectorFilterByStatus(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/ok", 200)
	addTestExchange(insp.manager, "tun-1", "GET", "/bad", 400)
	addTestExchange(insp.manager, "tun-1", "GET", "/err", 500)

	req := httptest.NewRequest("GET", "/api/requests/http?status=4xx", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Requests []exchangeListItem `json:"requests"`
		Total    int                `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Total)
	assert.Len(t, resp.Requests, 1)
	assert.Equal(t, 400, resp.Requests[0].StatusCode)
}

func TestInspectorStatus(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/", 200)

	req := httptest.NewRequest("GET", "/api/status", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "dev", resp["version"])
	assert.Equal(t, true, resp["inspect_enabled"])
	assert.Equal(t, float64(1), resp["total_exchanges"])
}

func TestInspectorCORS(t *testing.T) {
	insp := newTestInspector()

	req := httptest.NewRequest("OPTIONS", "/api/status", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestInspectorListTunnels(t *testing.T) {
	insp := newTestInspector()

	// Without tunnels set, should return empty list.
	req := httptest.NewRequest("GET", "/api/tunnels", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Tunnels []map[string]any `json:"tunnels"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp.Tunnels)
}

func TestInspectorFilterByExactStatus(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/a", 200)
	addTestExchange(insp.manager, "tun-1", "GET", "/b", 201)
	addTestExchange(insp.manager, "tun-1", "GET", "/c", 204)

	req := httptest.NewRequest("GET", "/api/requests/http?status=201", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	var resp struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Total)
}

func TestInspectorPagination(t *testing.T) {
	insp := newTestInspector()
	for j := 0; j < 10; j++ {
		addTestExchange(insp.manager, "tun-1", "GET", "/page", 200)
	}

	req := httptest.NewRequest("GET", "/api/requests/http?limit=3&offset=2", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	var resp struct {
		Requests []exchangeListItem `json:"requests"`
		Total    int                `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 10, resp.Total)
	assert.Len(t, resp.Requests, 3)
}

func TestMatchStatus(t *testing.T) {
	tests := []struct {
		code   int
		filter string
		want   bool
	}{
		{200, "2xx", true},
		{299, "2xx", true},
		{300, "2xx", false},
		{301, "3xx", true},
		{404, "4xx", true},
		{500, "5xx", true},
		{200, "200", true},
		{200, "201", false},
		{404, "404", true},
		{200, "invalid", false},
	}

	for _, tt := range tests {
		got := matchStatus(tt.code, tt.filter)
		assert.Equal(t, tt.want, got, "matchStatus(%d, %q)", tt.code, tt.filter)
	}
}
