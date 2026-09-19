package core

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// authReq builds a request the way a legitimate local client does: no browser
// Origin, and carrying the inspector's session token.
func authReq(insp *Inspector, method, target string, body io.Reader) *http.Request {
	r := localReq(method, target, body)
	r.Header.Set("Authorization", "Bearer "+insp.token)
	return r
}

// localReq builds an unauthenticated request addressed to the loopback
// listener (httptest defaults the Host to example.com, which the guard
// rejects as a rebound host).
func localReq(method, target string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, target, body)
	r.Host = "127.0.0.1:4040"
	return r
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

	req := authReq(insp, "GET", "/api/requests/http", nil)
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

	req := authReq(insp, "GET", "/api/requests/http/"+ex.ID, nil)
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

	req := authReq(insp, "GET", "/api/requests/http/nonexistent-id", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestInspectorDeleteExchanges(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/a", 200)
	addTestExchange(insp.manager, "tun-1", "POST", "/b", 201)

	req := authReq(insp, "DELETE", "/api/requests/http", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify cleared.
	listReq := authReq(insp, "GET", "/api/requests/http", nil)
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

	req := authReq(insp, "GET", "/api/requests/http/summary", nil)
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

	req := authReq(insp, "GET", "/api/requests/http?method=POST", nil)
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

	req := authReq(insp, "GET", "/api/requests/http?status=4xx", nil)
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

	req := authReq(insp, "GET", "/api/status", nil)
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "dev", resp["version"])
	assert.Equal(t, true, resp["inspect_enabled"])
	assert.Equal(t, float64(1), resp["total_exchanges"])
}

// TestInspectorNoCORS pins that the inspector never opts into cross-origin
// access: no ACAO header, and a preflight from a web page is refused so a
// cross-site POST replay / DELETE wipe cannot even be attempted.
func TestInspectorNoCORS(t *testing.T) {
	insp := newTestInspector()

	req := authReq(insp, "OPTIONS", "/api/status", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Methods"))
}

// TestInspectorGuard covers the three ways a request can be refused: a
// non-loopback Host (DNS rebinding), a cross-site Origin/Referer (web page),
// and a missing or wrong session token.
func TestInspectorGuard(t *testing.T) {
	tests := []struct {
		name   string
		method string
		mutate func(*http.Request)
		want   int
	}{
		{"missing token", "GET", func(r *http.Request) { r.Header.Del("Authorization") }, http.StatusUnauthorized},
		{"wrong token", "GET", func(r *http.Request) { r.Header.Set("Authorization", "Bearer nope") }, http.StatusUnauthorized},
		{"empty bearer", "GET", func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") }, http.StatusUnauthorized},
		{"cross-site origin", "GET", func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, http.StatusForbidden},
		{"cross-site referer", "GET", func(r *http.Request) { r.Header.Set("Referer", "https://evil.example/x") }, http.StatusForbidden},
		{"rebound host", "GET", func(r *http.Request) { r.Host = "evil.example" }, http.StatusForbidden},
		{"other local app origin", "GET", func(r *http.Request) { r.Header.Set("Origin", "http://127.0.0.1:3000") }, http.StatusForbidden},
		{"cross-site replay", "POST", func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, http.StatusForbidden},
		{"cross-site wipe", "DELETE", func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insp := newTestInspector()
			ex := addTestExchange(insp.manager, "tun-1", "GET", "/api/v1/users", 200)

			req := authReq(insp, tt.method, "/api/requests/http", nil)
			tt.mutate(req)
			rec := httptest.NewRecorder()
			insp.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
			assert.NotContains(t, rec.Body.String(), ex.ID, "captured traffic must not leak")
			assert.NotNil(t, insp.manager.GetOrCreate("tun-1").Get(ex.ID), "buffer must survive")
		})
	}
}

// TestInspectorEmptyTokenFailsClosed: an inspector without a session token
// rejects everything instead of serving traffic unauthenticated.
func TestInspectorEmptyTokenFailsClosed(t *testing.T) {
	insp := newTestInspector()
	insp.token = ""

	req := localReq("GET", "/api/requests/http", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestInspectorUICookie: the bundled UI is same-origin, gets its token as a
// SameSite=Strict cookie when the shell is served, and may use it on /api.
func TestInspectorUICookie(t *testing.T) {
	insp := newTestInspector()
	addTestExchange(insp.manager, "tun-1", "GET", "/api/v1/users", 200)

	rec := httptest.NewRecorder()
	insp.ServeHTTP(rec, localReq("GET", "/", nil))

	var session *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == inspectorCookieName {
			session = c
		}
	}
	require.NotNil(t, session, "UI shell must hand out the session cookie")
	assert.Equal(t, insp.token, session.Value)
	assert.True(t, session.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, session.SameSite)

	req := localReq("GET", "/api/requests/http", nil)
	req.Header.Set("Origin", "http://127.0.0.1:4040")
	req.AddCookie(session)
	apiRec := httptest.NewRecorder()
	insp.ServeHTTP(apiRec, req)

	assert.Equal(t, http.StatusOK, apiRec.Code)
}

func TestInspectorListTunnels(t *testing.T) {
	insp := newTestInspector()

	// Without tunnels set, should return empty list.
	req := authReq(insp, "GET", "/api/tunnels", nil)
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

	req := authReq(insp, "GET", "/api/requests/http?status=201", nil)
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

	req := authReq(insp, "GET", "/api/requests/http?limit=3&offset=2", nil)
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

// TestInspectorSharedTokenFile pins how legitimate same-machine clients get
// in: Start publishes the session token in ~/.fxtunnel/inspector.token (0600),
// and a request over the real socket is refused without it.
func TestInspectorSharedTokenFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	insp := newTestInspector()
	require.NoError(t, insp.Start(context.Background()))
	t.Cleanup(func() { _ = insp.Stop() })

	tokenPath := filepath.Join(home, ".fxtunnel", "inspector.token")
	data, err := os.ReadFile(tokenPath)
	require.NoError(t, err)
	assert.Equal(t, insp.token, strings.TrimSpace(string(data)))

	fi, err := os.Stat(tokenPath)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o600), fi.Mode().Perm())

	base := "http://" + insp.Addr()
	resp, err := http.Get(base + "/api/status") //nolint:noctx // short-lived test request
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	req, err := http.NewRequest("GET", base+"/api/status", nil) //nolint:noctx // short-lived test request
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(data)))
	authed, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer authed.Body.Close()
	assert.Equal(t, http.StatusOK, authed.StatusCode)
}
