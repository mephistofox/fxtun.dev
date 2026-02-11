package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/config"
	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/mephistofox/fxtunnel/internal/protocol"
)

func (s *Server) authenticate(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, authMsg *protocol.AuthMessage, log zerolog.Logger) (*Client, error) {
	// First, try to authenticate with database token (new system)
	if s.db != nil {
		tokenHash := hashToken(authMsg.Token)
		apiToken, err := s.db.Tokens.GetByTokenHash(tokenHash)
		if err == nil && apiToken != nil {
			// Check IP whitelist
			if !apiToken.IsIPAllowed(conn.RemoteAddr().String()) {
				result := &protocol.AuthResultMessage{
					Message: protocol.NewMessage(protocol.MsgAuthResult),
					Success: false,
					Error:   "IP not allowed",
					Code:    protocol.ErrCodePermissionDenied,
				}
				_ = codec.Encode(result)
				return nil, fmt.Errorf("IP not allowed for token")
			}

			// Valid DB token found
			client := s.createClientFromDBToken(conn, session, controlStream, codec, apiToken, log)
			client.SessionSecret = generateSessionSecret()

			// Update last used
			if err := s.db.Tokens.UpdateLastUsed(apiToken.ID); err != nil {
				log.Warn().Err(err).Int64("token_id", apiToken.ID).Msg("Failed to update token last used")
			}

			// Link user to client
			s.clientMgr.linkUserClient(apiToken.UserID, client.ID)

			// Compute effective max tunnels
			maxTunnels := apiToken.MaxTunnels
			if client.Plan != nil && !IsUnlimited(client.Plan.MaxTunnels) && client.Plan.MaxTunnels < maxTunnels {
				maxTunnels = client.Plan.MaxTunnels
			}

			// Send success
			result := &protocol.AuthResultMessage{
				Message:    protocol.NewMessage(protocol.MsgAuthResult),
				Success:    true,
				ClientID:   client.ID,
				MaxTunnels: maxTunnels,
				ServerName:    s.cfg.Domain.Base,
				SessionID:     client.ID,
				SessionSecret: client.SessionSecret,
				MinVersion:    s.cfg.Server.MinVersion,
			}
			if err := codec.Encode(result); err != nil {
				client.Close()
				return nil, fmt.Errorf("send auth result: %w", err)
			}

			log.Info().Int64("user_id", apiToken.UserID).Str("token_name", apiToken.Name).Msg("Authenticated with DB token")
			return client, nil
		}
	}

	// Try JWT authentication (for GUI login with phone/password)
	if s.authService != nil && isJWT(authMsg.Token) {
		claims, err := s.authService.ValidateAccessToken(authMsg.Token)
		if err != nil {
			// Check if token is expired - don't fallback to legacy tokens
			if err == auth.ErrTokenExpired {
				result := &protocol.AuthResultMessage{
					Message: protocol.NewMessage(protocol.MsgAuthResult),
					Success: false,
					Error:   "token expired",
					Code:    protocol.ErrCodeTokenExpired,
				}
				_ = codec.Encode(result)
				return nil, fmt.Errorf("token expired")
			}
			// Other JWT errors - continue to legacy token check
			log.Debug().Err(err).Msg("JWT validation failed, trying legacy tokens")
		} else if claims != nil {
			// Valid JWT - create client for user
			client := s.createClientFromJWT(conn, session, controlStream, codec, claims, log)
			client.SessionSecret = generateSessionSecret()

			// Link user to client
			s.clientMgr.linkUserClient(claims.UserID, client.ID)

			// Compute effective max tunnels for JWT auth
			maxTunnels := 10
			if client.Plan != nil && !IsUnlimited(client.Plan.MaxTunnels) {
				maxTunnels = client.Plan.MaxTunnels
			} else if client.Plan != nil && IsUnlimited(client.Plan.MaxTunnels) {
				maxTunnels = -1
			}

			// Send success
			result := &protocol.AuthResultMessage{
				Message:       protocol.NewMessage(protocol.MsgAuthResult),
				Success:       true,
				ClientID:      client.ID,
				MaxTunnels:    maxTunnels,
				ServerName:    s.cfg.Domain.Base,
				SessionID:     client.ID,
				SessionSecret: client.SessionSecret,
				MinVersion:    s.cfg.Server.MinVersion,
			}
			if err := codec.Encode(result); err != nil {
				client.Close()
				return nil, fmt.Errorf("send auth result: %w", err)
			}

			log.Info().Int64("user_id", claims.UserID).Str("phone", claims.Phone).Msg("Authenticated with JWT")
			return client, nil
		}
	}

	// Fallback: Check YAML config tokens (legacy system)
	if s.cfg.Auth.Enabled {
		tokenCfg := s.cfg.FindToken(authMsg.Token)
		if tokenCfg == nil {
			result := &protocol.AuthResultMessage{
				Message: protocol.NewMessage(protocol.MsgAuthResult),
				Success: false,
				Error:   "invalid token",
			}
			_ = codec.Encode(result)
			return nil, fmt.Errorf("invalid token")
		}

		// Create client with legacy token
		client := s.createClient(conn, session, controlStream, codec, tokenCfg, log)
		client.SessionSecret = generateSessionSecret()

		// Send success
		result := &protocol.AuthResultMessage{
			Message:       protocol.NewMessage(protocol.MsgAuthResult),
			Success:       true,
			ClientID:      client.ID,
			MaxTunnels:    tokenCfg.MaxTunnels,
			ServerName:    s.cfg.Domain.Base,
			SessionID:     client.ID,
			SessionSecret: client.SessionSecret,
			MinVersion:    s.cfg.Server.MinVersion,
		}
		if err := codec.Encode(result); err != nil {
			client.Close()
			return nil, fmt.Errorf("send auth result: %w", err)
		}

		return client, nil
	}

	// No auth required - create client without token
	client := s.createClient(conn, session, controlStream, codec, nil, log)
	client.SessionSecret = generateSessionSecret()

	result := &protocol.AuthResultMessage{
		Message:       protocol.NewMessage(protocol.MsgAuthResult),
		Success:       true,
		ClientID:      client.ID,
		MaxTunnels:    10, // Default limit
		ServerName:    s.cfg.Domain.Base,
		SessionID:     client.ID,
		SessionSecret: client.SessionSecret,
		MinVersion:    s.cfg.Server.MinVersion,
	}
	if err := codec.Encode(result); err != nil {
		client.Close()
		return nil, fmt.Errorf("send auth result: %w", err)
	}

	return client, nil
}

// createClientFromDBToken creates a client authenticated with a database token
func (s *Server) createClientFromDBToken(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, apiToken *database.APIToken, log zerolog.Logger) *Client {
	clientID := generateID()
	ctx, cancel := context.WithCancel(s.ctx)

	client := &Client{
		ID:           clientID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Token:        nil, // No legacy token
		Session:      session,
		ControlCodec: codec,
		ControlConn:  controlStream,
		Tunnels:      make(map[string]*Tunnel),
		Connected:    time.Now(),
		UserID:       apiToken.UserID,
		APITokenID:   apiToken.ID,
		DBToken:      apiToken,
		server:       s,
		conn:         conn,
		log:          log.With().Str("client_id", clientID).Int64("user_id", apiToken.UserID).Logger(),
		ctx:          ctx,
		cancel:       cancel,
	}
	client.lastPing.Store(time.Now().UnixNano())

	// Resolve admin status and plan from user record
	if s.db != nil && apiToken.UserID > 0 {
		if user, err := s.db.Users.GetByID(apiToken.UserID); err == nil && user != nil {
			client.IsAdmin = user.IsAdmin
			if user.PlanID > 0 {
				if plan, err := s.db.Plans.GetByID(user.PlanID); err == nil {
					client.Plan = plan
				}
			}
		}
	}

	s.initClientBandwidth(client)
	s.clientMgr.addClient(clientID, client)

	return client
}

// createClientFromJWT creates a client authenticated with a JWT token
func (s *Server) createClientFromJWT(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, claims *auth.Claims, log zerolog.Logger) *Client {
	clientID := generateID()
	ctx, cancel := context.WithCancel(s.ctx)

	client := &Client{
		ID:           clientID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Token:        nil, // No legacy token
		Session:      session,
		ControlCodec: codec,
		ControlConn:  controlStream,
		Tunnels:      make(map[string]*Tunnel),
		Connected:    time.Now(),
		UserID:       claims.UserID,
		IsAdmin:      claims.IsAdmin,
		server:       s,
		conn:         conn,
		log:          log.With().Str("client_id", clientID).Int64("user_id", claims.UserID).Logger(),
		ctx:          ctx,
		cancel:       cancel,
	}
	client.lastPing.Store(time.Now().UnixNano())

	// Load user plan
	if s.db != nil {
		if user, err := s.db.Users.GetByID(claims.UserID); err == nil && user != nil {
			if user.PlanID > 0 {
				if plan, err := s.db.Plans.GetByID(user.PlanID); err == nil {
					client.Plan = plan
				}
			}
		}
	}

	s.initClientBandwidth(client)
	s.clientMgr.addClient(clientID, client)

	return client
}

func (s *Server) createClient(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, token *config.TokenConfig, log zerolog.Logger) *Client {
	clientID := generateID()
	ctx, cancel := context.WithCancel(s.ctx)

	client := &Client{
		ID:           clientID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Token:        token,
		Session:      session,
		ControlCodec: codec,
		ControlConn:  controlStream,
		Tunnels:      make(map[string]*Tunnel),
		Connected:    time.Now(),
		server:       s,
		conn:         conn,
		log:          log.With().Str("client_id", clientID).Logger(),
		ctx:          ctx,
		cancel:       cancel,
	}
	client.lastPing.Store(time.Now().UnixNano())

	s.initClientBandwidth(client)
	s.clientMgr.addClient(clientID, client)

	return client
}

// initClientBandwidth initializes the bandwidth limiter for a client based on plan or default config.
func (s *Server) initClientBandwidth(client *Client) {
	mbps := s.cfg.Server.BandwidthLimitMbps

	// Plan-based override
	if client.Plan != nil && client.Plan.BandwidthMbps > 0 {
		mbps = client.Plan.BandwidthMbps
	}

	// Unlimited for admins or if explicitly set to 0
	if mbps <= 0 || client.IsAdmin {
		return
	}

	bytesPerSec := mbps * 1024 * 1024 / 8
	client.bwLimiter = NewBandwidthLimiter(bytesPerSec)
}

// generateSessionSecret creates a random secret for session pooling.
func generateSessionSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// isJWT checks if a token looks like a JWT (has 3 dot-separated parts)
func isJWT(token string) bool {
	if strings.HasPrefix(token, "sk_") {
		return false
	}
	parts := 0
	for _, c := range token {
		if c == '.' {
			parts++
		}
	}
	return parts == 2
}

// hashToken creates a SHA256 hash of a token for database lookup
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
