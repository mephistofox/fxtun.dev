package api

import (
	"fmt"
	"net/http"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/database"
)

// POST /api/auth/device/code
func (s *Server) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	session, err := s.deviceStore.Create()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to create device session")
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	authURL := fmt.Sprintf("%s://%s/auth/cli?session=%s", scheme, r.Host, session.ID)

	s.respondJSON(w, http.StatusOK, dto.DeviceCodeResponse{
		SessionID: session.ID,
		AuthURL:   authURL,
		ExpiresIn: 300,
	})
}

// GET /api/auth/device/token?session=XXX
func (s *Server) handleDevicePoll(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		s.respondError(w, http.StatusBadRequest, "session parameter required")
		return
	}

	session := s.deviceStore.Get(sessionID)
	if session == nil {
		s.respondError(w, http.StatusNotFound, "session not found")
		return
	}

	resp := dto.DevicePollResponse{
		Status: string(session.Status),
	}
	if session.Status == deviceStatusAuthorized {
		resp.Token = session.Token
		s.deviceStore.Delete(sessionID)
	}

	s.respondJSON(w, http.StatusOK, resp)
}

// POST /api/auth/device/authorize (authenticated)
func (s *Server) handleDeviceAuthorize(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.DeviceAuthorizeRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SessionID == "" {
		s.respondError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	session := s.deviceStore.Get(req.SessionID)
	if session == nil || session.Status != deviceStatusPending {
		s.respondError(w, http.StatusBadRequest, "invalid or expired session")
		return
	}

	plainToken, err := auth.GenerateAPIToken()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	tokenHash := auth.HashToken(plainToken)
	dbToken := &database.APIToken{
		UserID:            user.ID,
		TokenHash:         tokenHash,
		Name:              "CLI (device flow)",
		AllowedSubdomains: []string{"*"},
		MaxTunnels:        10,
	}

	if err := s.db.Tokens.Create(dbToken); err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, database.ActionTokenCreated,
		map[string]interface{}{
			"token_id":   dbToken.ID,
			"token_name": dbToken.Name,
			"method":     "device_flow",
		},
		ipAddress)

	s.deviceStore.Authorize(req.SessionID, plainToken)

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "authorized"})
}
