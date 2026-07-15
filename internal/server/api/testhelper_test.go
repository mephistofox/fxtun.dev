package api

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mephistofox/fxtun.dev/internal/server/auth"
	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/server/database"
	"github.com/rs/zerolog"
)

// mockTunnelProvider implements TunnelProvider for tests.
type mockTunnelProvider struct {
	tunnels     []TunnelInfo
	userTunnels map[int64][]TunnelInfo
	closeErr    error
	stats       Stats
}

func newMockTunnelProvider() *mockTunnelProvider {
	return &mockTunnelProvider{
		userTunnels: make(map[int64][]TunnelInfo),
	}
}

func (m *mockTunnelProvider) GetTunnelsByUserID(userID int64) []TunnelInfo {
	return m.userTunnels[userID]
}

func (m *mockTunnelProvider) CloseTunnelByID(tunnelID string, userID int64) error {
	return m.closeErr
}

func (m *mockTunnelProvider) GetStats() Stats {
	return m.stats
}

func (m *mockTunnelProvider) GetAllTunnels() []TunnelInfo {
	return m.tunnels
}

func (m *mockTunnelProvider) AdminCloseTunnel(tunnelID string) error {
	return m.closeErr
}

// testEnv holds all dependencies for API integration tests.
type testEnv struct {
	DB             *database.Database
	AuthService    *auth.Service
	TunnelProvider *mockTunnelProvider
	Server         *httptest.Server
	APIServer      *Server
}

// testDSN returns the PostgreSQL DSN for testing. It reads from TEST_DATABASE_DSN
// environment variable. If not set, the test is skipped.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping database-dependent test")
	}
	return dsn
}

// setupTestSchema creates an isolated PostgreSQL schema for a test and returns
// the DSN with search_path set to that schema. Cleans up after the test.
func setupTestSchema(t *testing.T, baseDSN string) string {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, baseDSN)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Create a unique schema for this test (safe: schemaName is constructed from UnixNano)
	schemaName := fmt.Sprintf("test_%d", time.Now().UnixNano())
	_, err = pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %q", schemaName))
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	pool.Close()

	t.Cleanup(func() {
		cleanPool, err := pgxpool.New(ctx, baseDSN)
		if err == nil {
			_, _ = cleanPool.Exec(ctx, fmt.Sprintf("DROP SCHEMA %q CASCADE", schemaName))
			cleanPool.Close()
		}
	})

	// Append search_path to DSN
	separator := "?"
	if strings.Contains(baseDSN, "?") {
		separator = "&"
	}
	return baseDSN + separator + "search_path=" + schemaName
}

// setupTestEnv creates a fully wired test environment with PostgreSQL,
// auth service, mock tunnel provider, and an httptest.Server ready for requests.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tmpDir := t.TempDir()

	baseDSN := testDSN(t)
	dbDSN := setupTestSchema(t, baseDSN)

	log := zerolog.New(os.Stderr).Level(zerolog.Disabled)

	db, err := database.New(dbDSN, log)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.ServerConfig{
		Server: config.ServerSettings{
			ControlPort: 4443,
			HTTPPort:    8080,
			TCPPortRange: config.PortRange{Min: 10000, Max: 20000},
			UDPPortRange: config.PortRange{Min: 20001, Max: 30000},
		},
		Domain: config.DomainSettings{
			Base:     "test.localhost",
			Wildcard: true,
		},
		Auth: config.AuthSettings{
			Enabled:                  true,
			JWTSecret:                "test-jwt-secret-at-least-32-chars-long!!",
			AccessTokenTTL:           "15m",
			RefreshTokenTTL:          "168h",
			MaxDomains:               3,
			PhoneRegistrationEnabled: true,
			PhoneRegistrationTarpit:  false,
		},
		Web: config.WebSettings{
			Enabled: false, // avoid validation requiring jwt_secret via Validate()
			Port:    8081,
			RateLimit: config.RateLimitConfig{
				Enabled: false,
			},
		},
		Database: config.DatabaseSettings{
			DSN: dbDSN,
		},
		TOTP: config.TOTPSettings{
			Enabled:       true,
			Issuer:        "fxTunnel-test",
			EncryptionKey: "test-totp-key-1234567890abcdef",
		},
		Downloads: config.DownloadsSettings{
			Enabled: false,
			Path:    filepath.Join(tmpDir, "downloads"),
		},
	}

	authSvc := auth.NewService(
		db,
		cfg.Auth.JWTSecret,
		15*time.Minute,
		168*time.Hour,
		cfg.TOTP.Issuer,
		[]byte(cfg.TOTP.EncryptionKey),
		cfg.Auth.MaxDomains,
		log,
	)

	tp := newMockTunnelProvider()

	apiServer := New(cfg, db, authSvc, tp, nil, nil, log)

	ts := httptest.NewServer(apiServer.Router())
	t.Cleanup(func() { ts.Close() })

	return &testEnv{
		DB:             db,
		AuthService:    authSvc,
		TunnelProvider: tp,
		Server:         ts,
		APIServer:      apiServer,
	}
}

// testUser holds a created user and their access token.
type testUser struct {
	User        *database.User
	AccessToken string
}

// createTestUser registers a user and returns the user along with a valid access token.
func (env *testEnv) createTestUser(t *testing.T, phone, password, displayName string) *testUser {
	t.Helper()

	// Register user through auth service
	user, tokenPair, err := env.AuthService.Register(phone, password, displayName, "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to register test user: %v", err)
	}

	return &testUser{
		User:        user,
		AccessToken: tokenPair.AccessToken,
	}
}

// createTestAdmin creates a test user and promotes them to admin.
func (env *testEnv) createTestAdmin(t *testing.T, phone, password, displayName string) *testUser {
	t.Helper()

	tu := env.createTestUser(t, phone, password, displayName)

	// Promote to admin directly in the database
	_, err := env.DB.Pool().Exec(context.Background(), "UPDATE users SET is_admin = TRUE WHERE id = $1", tu.User.ID)
	if err != nil {
		t.Fatalf("failed to promote user to admin: %v", err)
	}
	tu.User.IsAdmin = true

	// Re-login to get a token with admin claims
	_, tokenPair, err := env.AuthService.Login(phone, password, "", "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to login as admin: %v", err)
	}
	tu.AccessToken = tokenPair.AccessToken

	return tu
}
