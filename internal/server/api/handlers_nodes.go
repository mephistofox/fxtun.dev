package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mephistofox/fxtun.dev/internal/server/auth"
	"github.com/mephistofox/fxtun.dev/internal/server/database"
	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

// --- DTOs ---

type nodeRegisterRequest struct {
	Name       string `json:"name"`
	Region     string `json:"region"`
	PublicAddr string `json:"public_addr"`
	HTTPAddr   string `json:"http_addr"`
	Version    string `json:"version"`
}

type nodeRegisterResponse struct {
	NodeID string `json:"node_id"`
	Status string `json:"status"`
}

type nodeHeartbeatRequest struct {
	NodeID      string `json:"node_id"`
	TunnelCount int    `json:"tunnel_count"`
	ClientCount int    `json:"client_count"`
}

type verifyTokenRequest struct {
	Token string `json:"token"`
}

type verifyTokenResponse struct {
	Valid           bool   `json:"valid"`
	UserID          int64  `json:"user_id,omitempty"`
	MaxTunnels      int    `json:"max_tunnels,omitempty"`
	MaxDataSessions int    `json:"max_data_sessions,omitempty"`
	IsAdmin         bool   `json:"is_admin,omitempty"`
	InspectorEnabled bool  `json:"inspector_enabled,omitempty"`
	Error           string `json:"error,omitempty"`
}

type adminNodeDTO struct {
	ID              int64      `json:"id"`
	NodeID          string     `json:"node_id"`
	Name            string     `json:"name"`
	Region          string     `json:"region"`
	PublicAddr      string     `json:"public_addr"`
	HTTPAddr        string     `json:"http_addr"`
	Status          string     `json:"status"`
	Version         string     `json:"version"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// --- Middleware ---

// nodeTokenMiddleware validates the node hub_token from the Authorization header.
func (s *Server) nodeTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := s.cfg.Node.HubToken
		if expected == "" {
			http.Error(w, `{"error":"node token not configured"}`, http.StatusInternalServerError)
			return
		}

		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, `{"error":"missing node token"}`, http.StatusUnauthorized)
			return
		}

		// Constant-time comparison to prevent timing attacks
		expectedHash := sha256.Sum256([]byte(expected))
		actualHash := sha256.Sum256([]byte(token))
		if subtle.ConstantTimeCompare(expectedHash[:], actualHash[:]) != 1 {
			http.Error(w, `{"error":"invalid node token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return h[7:]
	}
	return ""
}

// --- Node Registration (called by nodes) ---

func (s *Server) handleNodeRegister(w http.ResponseWriter, r *http.Request) {
	var req nodeRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.PublicAddr == "" {
		http.Error(w, `{"error":"name and public_addr are required"}`, http.StatusBadRequest)
		return
	}

	// Check if node with this name already exists — reuse its node_id
	existing, err := s.db.EdgeNodes.GetByName(req.Name)
	if err == nil && existing != nil {
		// Update existing node's connection info
		existing.PublicAddr = req.PublicAddr
		existing.HTTPAddr = req.HTTPAddr
		existing.Region = req.Region
		existing.Version = req.Version
		_ = s.db.EdgeNodes.UpdateHeartbeat(existing.NodeID, existing.Metadata)

		// Re-add active node to Redis registry so hub can redirect clients to it
		if existing.Status == "active" && s.nodeRegistry != nil {
			_ = s.nodeRegistry.RegisterNode(nodeToStoreEntry(existing))
		}

		s.log.Info().
			Str("node_id", existing.NodeID).
			Str("name", req.Name).
			Str("status", existing.Status).
			Msg("Edge node re-registered (existing)")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodeRegisterResponse{
			NodeID: existing.NodeID,
			Status: existing.Status,
		})
		return
	}

	// New node — create with fresh UUID
	nodeID := uuid.New().String()
	node := &database.EdgeNode{
		NodeID:     nodeID,
		Name:       req.Name,
		Region:     req.Region,
		PublicAddr: req.PublicAddr,
		HTTPAddr:   req.HTTPAddr,
		Version:    req.Version,
		Metadata:   "{}",
	}

	if err := s.db.EdgeNodes.Create(node); err != nil {
		s.log.Error().Err(err).Msg("Failed to create edge node")
		http.Error(w, `{"error":"failed to register node"}`, http.StatusInternalServerError)
		return
	}

	s.log.Info().
		Str("node_id", nodeID).
		Str("name", req.Name).
		Str("region", req.Region).
		Msg("Edge node registered (pending approval)")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeRegisterResponse{
		NodeID: nodeID,
		Status: "pending",
	})
}

func (s *Server) handleNodeHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req nodeHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.NodeID == "" {
		http.Error(w, `{"error":"node_id is required"}`, http.StatusBadRequest)
		return
	}

	// Update heartbeat in DB
	metadata, _ := json.Marshal(map[string]int{
		"tunnel_count": req.TunnelCount,
		"client_count": req.ClientCount,
	})
	if err := s.db.EdgeNodes.UpdateHeartbeat(req.NodeID, string(metadata)); err != nil {
		if errors.Is(err, database.ErrEdgeNodeNotFound) {
			// Node might be pending approval — check and return status
			node, dbErr := s.db.EdgeNodes.GetByNodeID(req.NodeID)
			if dbErr == nil && node != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				json.NewEncoder(w).Encode(map[string]string{"status": node.Status})
				return
			}
			http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
			return
		}
		s.log.Error().Err(err).Str("node_id", req.NodeID).Msg("Failed to update node heartbeat")
		http.Error(w, `{"error":"heartbeat update failed"}`, http.StatusInternalServerError)
		return
	}

	// Update Redis node registry if available
	if s.nodeRegistry != nil {
		_ = s.nodeRegistry.HeartbeatNode(req.NodeID, req.TunnelCount, req.ClientCount)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleNodeTLSCert returns the hub's TLS certificate only to approved edge nodes.
func (s *Server) handleNodeTLSCert(w http.ResponseWriter, r *http.Request) {
	// Require node_id query param
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		http.Error(w, `{"error":"node_id is required"}`, http.StatusBadRequest)
		return
	}

	// Verify node exists and is approved
	node, err := s.db.EdgeNodes.GetByNodeID(nodeID)
	if err != nil || node == nil {
		http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
		return
	}
	if node.Status != "active" {
		s.log.Warn().Str("node_id", nodeID).Str("status", node.Status).Msg("Unapproved node tried to fetch TLS cert")
		http.Error(w, `{"error":"node not approved"}`, http.StatusForbidden)
		return
	}

	certFile := s.cfg.TLS.CertFile
	keyFile := s.cfg.TLS.KeyFile
	if certFile == "" || keyFile == "" {
		http.Error(w, `{"error":"TLS not configured on hub"}`, http.StatusNotFound)
		return
	}

	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to read TLS cert")
		http.Error(w, `{"error":"failed to read cert"}`, http.StatusInternalServerError)
		return
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to read TLS key")
		http.Error(w, `{"error":"failed to read key"}`, http.StatusInternalServerError)
		return
	}

	s.log.Info().Str("node_id", nodeID).Str("name", node.Name).Msg("TLS cert issued to approved node")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"cert_pem": string(certPEM),
		"key_pem":  string(keyPEM),
	})
}

// --- Internal Auth Verification (called by nodes to validate client tokens) ---

func (s *Server) handleVerifyClientToken(w http.ResponseWriter, r *http.Request) {
	var req verifyTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(verifyTokenResponse{Valid: false, Error: "empty token"})
		return
	}

	resp := s.verifyToken(req.Token)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) verifyToken(token string) verifyTokenResponse {
	// Try DB API token first
	tokenHash := sha256Hash(token)
	apiToken, err := s.db.Tokens.GetByTokenHash(tokenHash)
	if err == nil && apiToken != nil {
		user, err := s.db.Users.GetByID(apiToken.UserID)
		if err != nil {
			return verifyTokenResponse{Valid: false, Error: "user not found"}
		}
		if !user.IsActive {
			return verifyTokenResponse{Valid: false, Error: "user blocked"}
		}

		maxTunnels := apiToken.MaxTunnels
		maxDataSessions := 8
		inspectorEnabled := false

		if user.PlanID > 0 {
			plan, err := s.db.Plans.GetByID(user.PlanID)
			if err == nil {
				if maxTunnels == 0 {
					maxTunnels = plan.MaxTunnels
				}
				maxDataSessions = plan.MaxDataSessions
				inspectorEnabled = plan.InspectorEnabled
			}
		}

		return verifyTokenResponse{
			Valid:            true,
			UserID:           apiToken.UserID,
			MaxTunnels:       maxTunnels,
			MaxDataSessions:  maxDataSessions,
			IsAdmin:          user.IsAdmin,
			InspectorEnabled: inspectorEnabled,
		}
	}

	// Try JWT
	if s.authService != nil && isJWT(token) {
		claims, err := s.authService.ValidateAccessToken(token)
		if err != nil {
			return verifyTokenResponse{Valid: false, Error: "invalid token"}
		}

		maxTunnels := 10
		maxDataSessions := 8
		inspectorEnabled := false

		if claims.UserID > 0 {
			user, err := s.db.Users.GetByID(claims.UserID)
			if err == nil && user.PlanID > 0 {
				plan, err := s.db.Plans.GetByID(user.PlanID)
				if err == nil {
					maxTunnels = plan.MaxTunnels
					maxDataSessions = plan.MaxDataSessions
					inspectorEnabled = plan.InspectorEnabled
				}
			}
		}

		return verifyTokenResponse{
			Valid:            true,
			UserID:           claims.UserID,
			MaxTunnels:       maxTunnels,
			MaxDataSessions:  maxDataSessions,
			IsAdmin:          claims.IsAdmin,
			InspectorEnabled: inspectorEnabled,
		}
	}

	return verifyTokenResponse{Valid: false, Error: "invalid token"}
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

func isJWT(token string) bool {
	if strings.HasPrefix(token, "sk_") {
		return false
	}
	return strings.Count(token, ".") == 2
}

// --- Admin Node Management ---

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	nodes, err := s.db.EdgeNodes.List(status)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to list edge nodes")
		http.Error(w, `{"error":"failed to list nodes"}`, http.StatusInternalServerError)
		return
	}

	dtos := make([]adminNodeDTO, 0, len(nodes))
	for _, n := range nodes {
		dtos = append(dtos, adminNodeDTO{
			ID:              n.ID,
			NodeID:          n.NodeID,
			Name:            n.Name,
			Region:          n.Region,
			PublicAddr:      n.PublicAddr,
			HTTPAddr:        n.HTTPAddr,
			Status:          n.Status,
			Version:         n.Version,
			LastHeartbeatAt: n.LastHeartbeatAt,
			ApprovedAt:      n.ApprovedAt,
			CreatedAt:       n.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": dtos,
		"total": len(dtos),
	})
}

func (s *Server) handleApproveNode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid node id"}`, http.StatusBadRequest)
		return
	}

	adminUser := auth.GetUserFromContext(r.Context())
	if adminUser == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := s.db.EdgeNodes.UpdateStatus(id, "active", adminUser.ID); err != nil {
		if errors.Is(err, database.ErrEdgeNodeNotFound) {
			http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
			return
		}
		s.log.Error().Err(err).Int64("id", id).Msg("Failed to approve edge node")
		http.Error(w, `{"error":"failed to approve node"}`, http.StatusInternalServerError)
		return
	}

	// Register in Redis if available
	node, err := s.db.EdgeNodes.GetByID(id)
	if err == nil && s.nodeRegistry != nil {
		_ = s.nodeRegistry.RegisterNode(nodeToStoreEntry(node))
	}

	s.log.Info().Int64("id", id).Msg("Edge node approved")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "active"})
}

func (s *Server) handleDisableNode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid node id"}`, http.StatusBadRequest)
		return
	}

	adminUser := auth.GetUserFromContext(r.Context())
	if adminUser == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get node first for Redis cleanup
	node, _ := s.db.EdgeNodes.GetByID(id)

	if err := s.db.EdgeNodes.UpdateStatus(id, "disabled", adminUser.ID); err != nil {
		if errors.Is(err, database.ErrEdgeNodeNotFound) {
			http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to disable node"}`, http.StatusInternalServerError)
		return
	}

	// Remove from Redis
	if node != nil && s.nodeRegistry != nil {
		_ = s.nodeRegistry.UnregisterNode(node.NodeID)
	}

	s.log.Info().Int64("id", id).Msg("Edge node disabled")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "disabled"})
}

func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid node id"}`, http.StatusBadRequest)
		return
	}

	// Get node first for Redis cleanup
	node, _ := s.db.EdgeNodes.GetByID(id)

	if err := s.db.EdgeNodes.Delete(id); err != nil {
		if errors.Is(err, database.ErrEdgeNodeNotFound) {
			http.Error(w, `{"error":"node not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to delete node"}`, http.StatusInternalServerError)
		return
	}

	// Remove from Redis
	if node != nil && s.nodeRegistry != nil {
		_ = s.nodeRegistry.UnregisterNode(node.NodeID)
	}

	s.log.Info().Int64("id", id).Msg("Edge node deleted")
	w.WriteHeader(http.StatusNoContent)
}

// nodeToStoreEntry converts a DB EdgeNode to a store.NodeEntry.
func nodeToStoreEntry(n *database.EdgeNode) store.NodeEntry {
	return store.NodeEntry{
		NodeID:     n.NodeID,
		Name:       n.Name,
		Region:     n.Region,
		PublicAddr: n.PublicAddr,
		HTTPAddr:   n.HTTPAddr,
		Status:     "active",
	}
}
