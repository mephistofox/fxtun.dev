package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mephistofox/fxtun.dev/internal/server/api/dto"
	"github.com/mephistofox/fxtun.dev/internal/server/auth"
	"github.com/mephistofox/fxtun.dev/internal/server/database"
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
	// The URL carries no secret: the user must type the code shown in their
	// own terminal, which is what stops a stranger's session from being
	// approved through a link.
	authURL := fmt.Sprintf("%s://%s/auth/cli", scheme, r.Host)

	s.respondJSON(w, http.StatusOK, dto.DeviceCodeResponse{
		SessionID: session.ID,
		UserCode:  session.UserCode,
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
		Status: session.Status,
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

	userCode := strings.ToUpper(strings.TrimSpace(req.UserCode))
	if userCode == "" {
		s.respondError(w, http.StatusBadRequest, "user_code is required")
		return
	}

	session := s.deviceStore.GetByUserCode(userCode)
	if session == nil || session.Status != deviceStatusPending {
		s.respondError(w, http.StatusBadRequest, "invalid or expired code")
		return
	}

	// Determine plan limits. A missing plan means the default tier, never
	// "unlimited" — otherwise a user with a null plan_id bypasses the cap.
	maxTunnels := 10
	maxTokens := defaultMaxTokens

	if user.Plan != nil {
		maxTunnels = user.Plan.MaxTunnelsPerToken
		maxTokens = user.Plan.MaxTokens
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
		MaxTunnels:        maxTunnels,
	}

	// Same transactional limit check as the regular token endpoint: counting
	// first and inserting after let parallel requests all pass a stale count.
	if err := s.db.Tokens.CreateWithLimit(dbToken, maxTokens); err != nil {
		if errors.Is(err, database.ErrMaxTokensReached) {
			s.respondErrorWithCode(w, http.StatusForbidden, "MAX_TOKENS", "token limit reached for your plan")
			return
		}
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

	s.deviceStore.Authorize(session.ID, plainToken)

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "authorized"})
}
