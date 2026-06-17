package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/rs/zerolog"

	"github.com/mephistofox/fxtun.dev/internal/server/auth"
	"github.com/mephistofox/fxtun.dev/internal/config"
	"github.com/mephistofox/fxtun.dev/internal/server/database"
	"github.com/mephistofox/fxtun.dev/internal/server/geoip"
	"github.com/mephistofox/fxtun.dev/internal/protocol"
	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

// errRedirected is a sentinel error returned when the client is redirected to a node.
var errRedirected = errors.New("client redirected to edge node")

func (s *Server) authenticate(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, authMsg *protocol.AuthMessage, log zerolog.Logger) (*Client, error) {
	// Node mode: delegate token verification to hub
	if s.mode == config.ModeNode && s.hubClient != nil {
		return s.authenticateViaHub(conn, session, controlStream, codec, authMsg, log)
	}

	// Hub mode: check if we should redirect to an edge node
	// We do a lightweight token validation first, then redirect if valid
	if s.mode == config.ModeHub && s.nodeRegistry != nil {
		if redirected, err := s.tryRedirectToNode(conn, codec, authMsg, log); redirected {
			return nil, err
		}
	}

	// First, try to authenticate with database token (new system)
	if s.db != nil {
		tokenHash := hashToken(authMsg.Token)
		apiToken, err := s.db.Tokens.GetByTokenHash(tokenHash)
		if err != nil && !errors.Is(err, database.ErrTokenNotFound) {
			// Real database error → fail-closed, do not fall through to other auth methods
			log.Error().Err(err).Msg("Database error during token authentication")
			result := &protocol.AuthResultMessage{
				Message: protocol.NewMessage(protocol.MsgAuthResult),
				Success: false,
				Error:   "internal error",
				Code:    protocol.ErrCodeInternalError,
			}
			_ = codec.Encode(result)
			return nil, fmt.Errorf("database error during auth: %w", err)
		}
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
			client.SessionSecretExpiry = time.Now().Add(5 * time.Minute)

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
				Message:         protocol.NewMessage(protocol.MsgAuthResult),
				Success:         true,
				ClientID:        client.ID,
				MaxTunnels:      maxTunnels,
				MaxDataSessions: effectiveMaxDataSessions(client.Plan),
				ServerName:      s.cfg.Domain.Base,
				SessionID:       client.ID,
				SessionSecret:   client.SessionSecret,
				MinVersion:      s.cfg.Server.MinVersion,
				Capabilities:    buildCapabilities(client.Plan, client.IsAdmin),
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
			// JWT validation failed → fail-closed, do not fall through to legacy tokens
			log.Warn().Err(err).Msg("JWT validation failed")
			result := &protocol.AuthResultMessage{
				Message: protocol.NewMessage(protocol.MsgAuthResult),
				Success: false,
				Error:   "invalid token",
				Code:    protocol.ErrCodeAuthFailed,
			}
			_ = codec.Encode(result)
			return nil, fmt.Errorf("JWT validation failed: %w", err)
		} else if claims != nil {
			// Valid JWT - create client for user
			client := s.createClientFromJWT(conn, session, controlStream, codec, claims, log)
			client.SessionSecret = generateSessionSecret()
			client.SessionSecretExpiry = time.Now().Add(5 * time.Minute)

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
				Message:         protocol.NewMessage(protocol.MsgAuthResult),
				Success:         true,
				ClientID:        client.ID,
				MaxTunnels:      maxTunnels,
				MaxDataSessions: effectiveMaxDataSessions(client.Plan),
				ServerName:      s.cfg.Domain.Base,
				SessionID:       client.ID,
				SessionSecret:   client.SessionSecret,
				MinVersion:      s.cfg.Server.MinVersion,
				Capabilities:    buildCapabilities(client.Plan, client.IsAdmin),
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
		client.SessionSecretExpiry = time.Now().Add(5 * time.Minute)

		// Send success
		result := &protocol.AuthResultMessage{
			Message:         protocol.NewMessage(protocol.MsgAuthResult),
			Success:         true,
			ClientID:        client.ID,
			MaxTunnels:      tokenCfg.MaxTunnels,
			MaxDataSessions: effectiveMaxDataSessions(client.Plan),
			ServerName:      s.cfg.Domain.Base,
			SessionID:       client.ID,
			SessionSecret:   client.SessionSecret,
			MinVersion:      s.cfg.Server.MinVersion,
			Capabilities:    buildCapabilities(client.Plan, client.IsAdmin),
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
	client.SessionSecretExpiry = time.Now().Add(5 * time.Minute)

	result := &protocol.AuthResultMessage{
		Message:         protocol.NewMessage(protocol.MsgAuthResult),
		Success:         true,
		ClientID:        client.ID,
		MaxTunnels:      10, // Default limit
		MaxDataSessions: effectiveMaxDataSessions(client.Plan),
		ServerName:      s.cfg.Domain.Base,
		SessionID:       client.ID,
		SessionSecret:   client.SessionSecret,
		MinVersion:      s.cfg.Server.MinVersion,
		Capabilities:    buildCapabilities(client.Plan, client.IsAdmin),
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

	s.clientMgr.addClient(clientID, client)

	return client
}

// generateSessionSecret creates a random secret for session pooling.
func generateSessionSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
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

const defaultMaxDataSessions = 16

// effectiveMaxDataSessions returns the max data sessions from plan.
// Returns 0 for unlimited (admin), defaultMaxDataSessions if no plan.
func effectiveMaxDataSessions(plan *database.Plan) int {
	if plan == nil {
		return defaultMaxDataSessions
	}
	if IsUnlimited(plan.MaxDataSessions) {
		return 0 // 0 = unlimited for client
	}
	if plan.MaxDataSessions > 0 {
		return plan.MaxDataSessions
	}
	return defaultMaxDataSessions
}

// buildCapabilities creates ClientCapabilities from the user's plan.
// Admin users always get full capabilities regardless of plan.
// Returns nil if no plan is set and user is not admin (legacy tokens).
func buildCapabilities(plan *database.Plan, isAdmin bool) *protocol.ClientCapabilities {
	if plan == nil && !isAdmin {
		return nil
	}
	caps := &protocol.ClientCapabilities{}
	if plan != nil {
		caps.InspectorEnabled = plan.InspectorEnabled
	}
	if isAdmin {
		caps.InspectorEnabled = true
	}
	return caps
}

// tryRedirectToNode checks if the client should be redirected to an edge node.
// Returns (true, errRedirected) if redirected, (false, nil) otherwise.
func (s *Server) tryRedirectToNode(conn net.Conn, codec *protocol.Codec, authMsg *protocol.AuthMessage, log zerolog.Logger) (bool, error) {
	// Quick token validation: check that token is valid before redirecting
	tokenValid := false
	if s.db != nil {
		tokenHash := hashToken(authMsg.Token)
		apiToken, err := s.db.Tokens.GetByTokenHash(tokenHash)
		if err == nil && apiToken != nil && apiToken.IsIPAllowed(conn.RemoteAddr().String()) {
			tokenValid = true
		}
	}
	if !tokenValid && s.authService != nil && isJWT(authMsg.Token) {
		_, err := s.authService.ValidateAccessToken(authMsg.Token)
		if err == nil {
			tokenValid = true
		}
	}
	if !tokenValid {
		// Token is invalid — let normal auth flow handle the error
		return false, nil
	}

	clientIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	candidates, selection := s.selectCandidates(clientIP)
	if len(candidates) == 0 {
		// No nodes available — hub handles the client itself
		return false, nil
	}

	primary := &candidates[0]

	// Don't redirect if selected node is THIS server — serve client locally
	if primary.NodeID == s.localNodeID || primary.Name == s.cfg.Node.Name {
		log.Info().
			Str("node", primary.Name).
			Str("selection", selection).
			Msg("Selected local node — serving client without redirect")
		return false, nil
	}

	redirectCandidates := make([]protocol.NodeRedirectCandidate, 0, len(candidates))
	for i := range candidates {
		redirectCandidates = append(redirectCandidates, protocol.NodeRedirectCandidate{
			Addr:   candidates[i].PublicAddr,
			NodeID: candidates[i].Name,
			Region: candidates[i].Region,
		})
	}

	result := &protocol.AuthResultMessage{
		Message:            protocol.NewMessage(protocol.MsgAuthResult),
		Success:            true,
		Code:               protocol.ErrCodeRedirect,
		RedirectAddr:       primary.PublicAddr,
		RedirectNodeID:     primary.Name,
		RedirectRegion:     primary.Region,
		RedirectCandidates: redirectCandidates,
	}
	if err := codec.Encode(result); err != nil {
		return false, fmt.Errorf("send redirect: %w", err)
	}

	country := ""
	if s.geoIP != nil {
		country = s.geoIP.Country(clientIP)
	}

	log.Info().
		Str("node", primary.Name).
		Str("region", primary.Region).
		Str("addr", primary.PublicAddr).
		Str("client_country", country).
		Str("selection", selection).
		Int("candidates", len(redirectCandidates)).
		Msg("Redirecting client to edge node")

	return true, errRedirected
}

// maxRedirectCandidates caps how many nodes are returned to the client for
// latency probing. Client-side parallel probes are cheap, but we keep this
// bounded to avoid pathological fan-out.
const maxRedirectCandidates = 5

// selectCandidates picks up to maxRedirectCandidates edge nodes for a client,
// ordered by (geo-match first, then TunnelCount ascending).
// Returns the selected nodes and the selection reason ("geo" or "least-loaded").
func (s *Server) selectCandidates(clientIP string) ([]store.NodeEntry, string) {
	nodes, err := s.nodeRegistry.ListActiveNodes()
	if err != nil || len(nodes) == 0 {
		return nil, ""
	}

	// Try GeoIP-based selection first: collect geo-matching nodes, sorted by load
	if s.geoIP != nil {
		country := s.geoIP.Country(clientIP)
		if country != "" {
			var matched []store.NodeEntry
			for i := range nodes {
				if geoip.RegionMatchesCountry(nodes[i].Region, country) {
					matched = append(matched, nodes[i])
				}
			}
			if len(matched) > 0 {
				sortNodesByLoad(matched)
				if len(matched) > maxRedirectCandidates {
					matched = matched[:maxRedirectCandidates]
				}
				return matched, "geo"
			}
		}
	}

	// Fallback: least-loaded across all nodes
	all := make([]store.NodeEntry, len(nodes))
	copy(all, nodes)
	sortNodesByLoad(all)
	if len(all) > maxRedirectCandidates {
		all = all[:maxRedirectCandidates]
	}
	return all, "least-loaded"
}

// sortNodesByLoad sorts nodes in-place by TunnelCount ascending (least-loaded first).
func sortNodesByLoad(nodes []store.NodeEntry) {
	// insertion sort — candidate list is tiny (<= a few dozen typically)
	for i := 1; i < len(nodes); i++ {
		j := i
		for j > 0 && nodes[j].TunnelCount < nodes[j-1].TunnelCount {
			nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
			j--
		}
	}
}

// selectBestNode picks the best edge node for a client (wrapper around
// selectCandidates for backward compatibility).
// Returns the selected node and the selection reason ("geo" or "least-loaded").
func (s *Server) selectBestNode(clientIP string) (*store.NodeEntry, string) {
	candidates, selection := s.selectCandidates(clientIP)
	if len(candidates) == 0 {
		return nil, ""
	}
	return &candidates[0], selection
}

// authenticateViaHub delegates client authentication to the hub (used in node mode).
func (s *Server) authenticateViaHub(conn net.Conn, session *yamux.Session, controlStream net.Conn, codec *protocol.Codec, authMsg *protocol.AuthMessage, log zerolog.Logger) (*Client, error) {
	info, err := s.hubClient.VerifyClientToken(authMsg.Token)
	if err != nil {
		log.Warn().Err(err).Msg("Hub token verification failed")
		result := &protocol.AuthResultMessage{
			Message: protocol.NewMessage(protocol.MsgAuthResult),
			Success: false,
			Error:   "authentication failed",
			Code:    protocol.ErrCodeAuthFailed,
		}
		_ = codec.Encode(result)
		return nil, fmt.Errorf("hub auth verification: %w", err)
	}

	if !info.Valid {
		result := &protocol.AuthResultMessage{
			Message: protocol.NewMessage(protocol.MsgAuthResult),
			Success: false,
			Error:   "invalid token",
			Code:    protocol.ErrCodeAuthFailed,
		}
		_ = codec.Encode(result)
		return nil, fmt.Errorf("hub rejected token")
	}

	clientID := generateID()
	ctx, cancel := context.WithCancel(s.ctx)
	client := &Client{
		ID:           clientID,
		RemoteAddr:   conn.RemoteAddr().String(),
		Session:      session,
		ControlCodec: codec,
		ControlConn:  controlStream,
		Tunnels:      make(map[string]*Tunnel),
		Connected:    time.Now(),
		UserID:       info.UserID,
		IsAdmin:      info.IsAdmin,
		server:       s,
		conn:         conn,
		log:          log.With().Str("client_id", clientID).Logger(),
		ctx:          ctx,
		cancel:       cancel,
	}
	client.lastPing.Store(time.Now().UnixNano())
	client.SessionSecret = generateSessionSecret()
	client.SessionSecretExpiry = time.Now().Add(5 * time.Minute)

	maxTunnels := info.MaxTunnels
	if maxTunnels == 0 {
		maxTunnels = 10
	}

	result := &protocol.AuthResultMessage{
		Message:         protocol.NewMessage(protocol.MsgAuthResult),
		Success:         true,
		ClientID:        clientID,
		MaxTunnels:      maxTunnels,
		MaxDataSessions: info.MaxDataSessions,
		ServerName:      s.cfg.Domain.Base,
		SessionID:       clientID,
		SessionSecret:   client.SessionSecret,
		MinVersion:      s.cfg.Server.MinVersion,
		Capabilities: &protocol.ClientCapabilities{
			InspectorEnabled: info.InspectorEnabled,
		},
	}
	if err := codec.Encode(result); err != nil {
		cancel()
		return nil, fmt.Errorf("send auth result: %w", err)
	}

	s.clientMgr.addClient(clientID, client)
	s.clientMgr.linkUserClient(info.UserID, clientID)
	log.Info().Int64("user_id", info.UserID).Msg("Authenticated via hub")
	return client, nil
}
