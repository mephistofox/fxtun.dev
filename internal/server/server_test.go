package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/protocol"
	"github.com/mephistofox/fxtunnel/internal/transport"
)

// testSetup creates a server with a temp SQLite database and a DB API token.
// Returns the server, database, and the raw token string.
func testSetup(t *testing.T) (*Server, *database.Database, string) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	log := zerolog.New(os.Stderr).Level(zerolog.Disabled)

	db, err := database.New(dbPath, log)
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.ServerConfig{
		Server: config.ServerSettings{
			ControlPort: 14443,
			HTTPPort:    18080,
			TCPPortRange: config.PortRange{
				Min: 30000,
				Max: 31000,
			},
			UDPPortRange: config.PortRange{
				Min: 31001,
				Max: 32000,
			},
		},
		Domain: config.DomainSettings{
			Base:     "test.local",
			Wildcard: true,
		},
		Auth: config.AuthSettings{
			Enabled: true,
		},
	}

	srv := New(cfg, log)
	srv.SetDatabase(db)

	// Create a user for the API token foreign key
	user := &database.User{
		Phone:        "+10000000000",
		PasswordHash: "fakehash",
		IsActive:     true,
	}
	if err := db.Users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create API token
	rawToken := "sk_test_integration_token_12345"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	apiToken := &database.APIToken{
		UserID:            user.ID,
		TokenHash:         tokenHash,
		Name:              "test-token",
		AllowedSubdomains: []string{"*"},
		MaxTunnels:        10,
		AllowedIPs:        []string{},
	}
	if err := db.Tokens.Create(apiToken); err != nil {
		t.Fatalf("create api token: %v", err)
	}

	return srv, db, rawToken
}

// dialServer creates a net.Pipe, wraps the server side via handleControlConnection
// and the client side in a transport.Session.
// Returns the client-side session.
func dialServer(t *testing.T, srv *Server) transport.Session {
	t.Helper()

	clientConn, serverConn := net.Pipe()

	// The server's handleControlConnection runs in a goroutine and calls wg.Done,
	// so we must increment wg first.
	srv.wg.Add(1)
	go srv.handleControlConnection(serverConn)

	// Perform compression handshake (client side, no compression in tests)
	rwc, _, err := protocol.NegotiateCompression(clientConn, false, false)
	if err != nil {
		t.Fatalf("NegotiateCompression: %v", err)
	}

	session, err := transport.NewYamuxSessionWithConfig(rwc, false, transport.YamuxConfig{
		MaxStreamWindowSize:    4 * 1024 * 1024,
		KeepAliveInterval:      24 * time.Hour,
		ConnectionWriteTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("transport.NewYamuxSessionWithConfig: %v", err)
	}

	return session
}

// openControlStream opens the first stream (control channel) and
// returns a protocol.Codec wrapping it.
func openControlStream(t *testing.T, session transport.Session) (*protocol.Codec, transport.Stream) {
	t.Helper()

	stream, err := session.OpenStream(context.Background())
	if err != nil {
		t.Fatalf("session.OpenStream: %v", err)
	}

	codec := protocol.NewCodec(stream, stream)
	return codec, stream
}

// sendAuth sends an AuthMessage and reads the AuthResultMessage.
func sendAuth(t *testing.T, codec *protocol.Codec, token string) *protocol.AuthResultMessage {
	t.Helper()

	authMsg := &protocol.AuthMessage{
		Message: protocol.NewMessage(protocol.MsgAuth),
		Token:   token,
	}
	if err := codec.Encode(authMsg); err != nil {
		t.Fatalf("encode auth: %v", err)
	}

	var result protocol.AuthResultMessage
	if err := codec.Decode(&result); err != nil {
		t.Fatalf("decode auth result: %v", err)
	}
	return &result
}

func TestServerValidAuth(t *testing.T) {
	srv, _, rawToken := testSetup(t)
	defer srv.cancel()

	session := dialServer(t, srv)
	defer session.Close()

	codec, _ := openControlStream(t, session)
	result := sendAuth(t, codec, rawToken)

	if !result.Success {
		t.Fatalf("expected auth success, got error: %s", result.Error)
	}
	if result.ClientID == "" {
		t.Fatal("expected non-empty client ID")
	}
	if result.ServerName != "test.local" {
		t.Fatalf("expected server name 'test.local', got %q", result.ServerName)
	}
	if result.MaxTunnels != 10 {
		t.Fatalf("expected max_tunnels=10, got %d", result.MaxTunnels)
	}

	// Verify client is registered on the server
	client := srv.GetClient(result.ClientID)
	if client == nil {
		t.Fatal("client not found on server after auth")
	}
}

func TestServerInvalidToken(t *testing.T) {
	srv, _, _ := testSetup(t)
	defer srv.cancel()

	session := dialServer(t, srv)
	defer session.Close()

	codec, _ := openControlStream(t, session)
	result := sendAuth(t, codec, "sk_invalid_token_that_does_not_exist")

	if result.Success {
		t.Fatal("expected auth failure for invalid token")
	}
	if result.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestServerHTTPTunnelCreation(t *testing.T) {
	srv, _, rawToken := testSetup(t)
	defer srv.cancel()

	session := dialServer(t, srv)
	defer session.Close()

	codec, _ := openControlStream(t, session)
	result := sendAuth(t, codec, rawToken)
	if !result.Success {
		t.Fatalf("auth failed: %s", result.Error)
	}

	// Request an HTTP tunnel
	tunnelReq := &protocol.TunnelRequestMessage{
		Message:    protocol.NewMessage(protocol.MsgTunnelRequest),
		TunnelType: protocol.TunnelHTTP,
		Name:       "my-web-app",
		Subdomain:  "myapp",
		LocalPort:  3000,
	}
	tunnelReq.RequestID = "req-1"
	if err := codec.Encode(tunnelReq); err != nil {
		t.Fatalf("encode tunnel request: %v", err)
	}

	// Read tunnel created response
	var created protocol.TunnelCreatedMessage
	if err := codec.Decode(&created); err != nil {
		t.Fatalf("decode tunnel created: %v", err)
	}

	if created.Type != protocol.MsgTunnelCreated {
		t.Fatalf("expected message type %s, got %s", protocol.MsgTunnelCreated, created.Type)
	}
	if created.TunnelID == "" {
		t.Fatal("expected non-empty tunnel ID")
	}
	if created.TunnelType != protocol.TunnelHTTP {
		t.Fatalf("expected tunnel type http, got %s", created.TunnelType)
	}
	if created.Subdomain != "myapp" {
		t.Fatalf("expected subdomain 'myapp', got %q", created.Subdomain)
	}
	if created.URL != "http://myapp.test.local" {
		t.Fatalf("expected URL 'http://myapp.test.local', got %q", created.URL)
	}

	// Verify tunnel is registered on the server
	stats := srv.GetStats()
	if stats.ActiveTunnels != 1 {
		t.Fatalf("expected 1 active tunnel, got %d", stats.ActiveTunnels)
	}
	if stats.HTTPTunnels != 1 {
		t.Fatalf("expected 1 HTTP tunnel, got %d", stats.HTTPTunnels)
	}
}

func TestServerTunnelClose(t *testing.T) {
	srv, _, rawToken := testSetup(t)
	defer srv.cancel()

	session := dialServer(t, srv)
	defer session.Close()

	codec, _ := openControlStream(t, session)
	result := sendAuth(t, codec, rawToken)
	if !result.Success {
		t.Fatalf("auth failed: %s", result.Error)
	}

	// Create tunnel first
	tunnelReq := &protocol.TunnelRequestMessage{
		Message:    protocol.NewMessage(protocol.MsgTunnelRequest),
		TunnelType: protocol.TunnelHTTP,
		Name:       "close-test",
		Subdomain:  "closeme",
		LocalPort:  8080,
	}
	tunnelReq.RequestID = "req-close"
	if err := codec.Encode(tunnelReq); err != nil {
		t.Fatalf("encode tunnel request: %v", err)
	}

	var created protocol.TunnelCreatedMessage
	if err := codec.Decode(&created); err != nil {
		t.Fatalf("decode tunnel created: %v", err)
	}
	if created.TunnelID == "" {
		t.Fatal("expected non-empty tunnel ID")
	}

	// Now close the tunnel
	closeMsg := &protocol.TunnelCloseMessage{
		Message:  protocol.NewMessage(protocol.MsgTunnelClose),
		TunnelID: created.TunnelID,
	}
	if err := codec.Encode(closeMsg); err != nil {
		t.Fatalf("encode tunnel close: %v", err)
	}

	// Read tunnel closed confirmation
	var closed protocol.TunnelClosedMessage
	if err := codec.Decode(&closed); err != nil {
		t.Fatalf("decode tunnel closed: %v", err)
	}

	if closed.Type != protocol.MsgTunnelClosed {
		t.Fatalf("expected message type %s, got %s", protocol.MsgTunnelClosed, closed.Type)
	}
	if closed.TunnelID != created.TunnelID {
		t.Fatalf("expected tunnel ID %s, got %s", created.TunnelID, closed.TunnelID)
	}

	// Verify tunnel is removed
	// Give a moment for cleanup
	time.Sleep(50 * time.Millisecond)
	stats := srv.GetStats()
	if stats.ActiveTunnels != 0 {
		t.Fatalf("expected 0 active tunnels after close, got %d", stats.ActiveTunnels)
	}
}

func TestServerPingPong(t *testing.T) {
	srv, _, rawToken := testSetup(t)
	defer srv.cancel()

	session := dialServer(t, srv)
	defer session.Close()

	codec, _ := openControlStream(t, session)
	result := sendAuth(t, codec, rawToken)
	if !result.Success {
		t.Fatalf("auth failed: %s", result.Error)
	}

	// Send ping
	ping := &protocol.PingMessage{
		Message: protocol.NewMessage(protocol.MsgPing),
	}
	if err := codec.Encode(ping); err != nil {
		t.Fatalf("encode ping: %v", err)
	}

	// Read pong
	var pong protocol.PongMessage
	if err := codec.Decode(&pong); err != nil {
		t.Fatalf("decode pong: %v", err)
	}

	if pong.Type != protocol.MsgPong {
		t.Fatalf("expected message type %s, got %s", protocol.MsgPong, pong.Type)
	}
}
