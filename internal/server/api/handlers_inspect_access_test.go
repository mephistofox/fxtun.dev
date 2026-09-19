package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/mephistofox/fxtunnel/internal/inspect"
	"github.com/stretchr/testify/require"
)

// mockReplayProvider records the last replayed request.
type mockReplayProvider struct {
	called bool
}

func (m *mockReplayProvider) ReplayRequest(subdomain string, req *http.Request) (*inspect.ReplayResult, error) {
	m.called = true
	return &inspect.ReplayResult{StatusCode: 200, Headers: http.Header{}, Body: []byte("ok")}, nil
}

// setupInspectEnv wires a DB-backed inspect manager with no live buffers, so all
// lookups go through the persisted (DB) path.
func setupInspectEnv(t *testing.T) *testEnv {
	t.Helper()
	env := setupTestEnv(t)
	mgr := inspect.NewManager(0, 4096)
	mgr.SetStore(env.DB.Exchanges)
	t.Cleanup(mgr.Close)
	env.APIServer.inspectProvider = mgr
	return env
}

// enableInspector moves the user onto a plan with inspector_enabled = TRUE and
// returns a fresh access token carrying that plan.
func (env *testEnv) enableInspector(t *testing.T, phone, password string) string {
	t.Helper()
	_, err := env.DB.Pool().Exec(t.Context(),
		`UPDATE users SET plan_id = (SELECT id FROM plans WHERE slug = 'pro') WHERE phone = $1`, phone)
	require.NoError(t, err)

	_, pair, err := env.AuthService.Login(phone, password, "", "test-agent", "127.0.0.1")
	require.NoError(t, err)
	return pair.AccessToken
}

func saveExchange(t *testing.T, env *testEnv, id, tunnelID string, userID int64) {
	t.Helper()
	err := env.DB.Exchanges.Save(&inspect.CapturedExchange{
		ID:             id,
		TunnelID:       tunnelID,
		Timestamp:      time.Now(),
		Method:         http.MethodPost,
		Path:           "/secret",
		Host:           "victim.test.localhost",
		RequestHeaders: http.Header{"Authorization": []string{"Bearer victim-secret"}},
		RequestBody:    []byte(`{"card":"4111111111111111"}`),
		StatusCode:     200,
	}, userID)
	require.NoError(t, err)
}

// TestGetExchange_CrossTenantDenied: knowing another user's exchange ID must not
// expose its request/response through a tunnel the caller does own.
func TestGetExchange_CrossTenantDenied(t *testing.T) {
	env := setupInspectEnv(t)

	victim := env.createTestUser(t, "+20000000801", "password123", "Victim")
	attacker := env.createTestUser(t, "+20000000802", "password123", "Attacker")
	attackerToken := env.enableInspector(t, "+20000000802", "password123")

	saveExchange(t, env, "ex-victim-1", "tun-victim", victim.User.ID)

	env.TunnelProvider.userTunnels[attacker.User.ID] = []TunnelInfo{
		{ID: "tun-attacker", Subdomain: "attacker", UserID: attacker.User.ID},
	}

	req, err := http.NewRequest(http.MethodGet,
		env.Server.URL+"/api/tunnels/tun-attacker/inspect/ex-victim-1", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+attackerToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"another user's exchange must not be readable")
}

// TestGetExchange_OwnExchangeStillReadable guards against over-tightening: a user's
// own exchange from an earlier (already closed) tunnel session stays readable.
func TestGetExchange_OwnExchangeReadable(t *testing.T) {
	env := setupInspectEnv(t)

	user := env.createTestUser(t, "+20000000803", "password123", "Owner")
	token := env.enableInspector(t, "+20000000803", "password123")

	saveExchange(t, env, "ex-own-1", "tun-old-session", user.User.ID)

	env.TunnelProvider.userTunnels[user.User.ID] = []TunnelInfo{
		{ID: "tun-current", Subdomain: "owner", UserID: user.User.ID},
	}

	req, err := http.NewRequest(http.MethodGet,
		env.Server.URL+"/api/tunnels/tun-current/inspect/ex-own-1", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestReplayExchange_CrossTenantDenied: replaying another user's exchange into an
// own tunnel would exfiltrate its headers and body.
func TestReplayExchange_CrossTenantDenied(t *testing.T) {
	env := setupInspectEnv(t)

	victim := env.createTestUser(t, "+20000000804", "password123", "Victim2")
	attacker := env.createTestUser(t, "+20000000805", "password123", "Attacker2")
	attackerToken := env.enableInspector(t, "+20000000805", "password123")

	saveExchange(t, env, "ex-victim-2", "tun-victim2", victim.User.ID)

	env.TunnelProvider.userTunnels[attacker.User.ID] = []TunnelInfo{
		{ID: "tun-attacker2", Subdomain: "attacker2", UserID: attacker.User.ID},
	}
	replay := &mockReplayProvider{}
	env.APIServer.replayProvider = replay

	req, err := http.NewRequest(http.MethodPost,
		env.Server.URL+"/api/tunnels/tun-attacker2/inspect/ex-victim-2/replay", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+attackerToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.False(t, replay.called, "another user's exchange must never be replayed")
}
