package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
	"github.com/mephistofox/fxtunnel/internal/database"
)

// handleListTokens returns the user's API tokens
func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tokens, err := s.db.Tokens.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get tokens")
		s.respondError(w, http.StatusInternalServerError, "failed to get tokens")
		return
	}

	tokenDTOs := make([]*dto.TokenDTO, len(tokens))
	for i, t := range tokens {
		tokenDTOs[i] = dto.TokenFromModel(t)
	}

	s.respondJSON(w, http.StatusOK, dto.TokensListResponse{
		Tokens: tokenDTOs,
		Total:  len(tokenDTOs),
	})
}

// handleCreateToken creates a new API token
func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateTokenRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Set defaults
	if len(req.AllowedSubdomains) == 0 {
		req.AllowedSubdomains = []string{"*"}
	}
	if req.MaxTunnels <= 0 {
		req.MaxTunnels = 10
	}
	if req.MaxTunnels > 100 {
		req.MaxTunnels = 100
	}

	// Generate token
	plainToken, err := auth.GenerateAPIToken()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate token")
		s.respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Hash token for storage
	tokenHash := auth.HashToken(plainToken)

	// Create token in database
	token := &database.APIToken{
		UserID:            user.ID,
		TokenHash:         tokenHash,
		Name:              req.Name,
		AllowedSubdomains: req.AllowedSubdomains,
		AllowedIPs:        req.AllowedIPs,
		MaxTunnels:        req.MaxTunnels,
	}

	if err := s.db.Tokens.Create(token); err != nil {
		s.log.Error().Err(err).Msg("Failed to create token")
		s.respondError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, database.ActionTokenCreated, map[string]interface{}{
		"token_id":   token.ID,
		"token_name": token.Name,
	}, ipAddress)

	// Return the plain token - this is the only time it will be shown!
	s.respondJSON(w, http.StatusCreated, dto.CreateTokenResponse{
		Token: plainToken,
		Info:  dto.TokenFromModel(token),
	})
}

// handleDeleteToken deletes an API token
func (s *Server) handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid token id")
		return
	}

	// Get token to verify ownership
	token, err := s.db.Tokens.GetByID(id)
	if err != nil {
		if errors.Is(err, database.ErrTokenNotFound) {
			s.respondError(w, http.StatusNotFound, "token not found")
			return
		}
		s.log.Error().Err(err).Msg("Failed to get token")
		s.respondError(w, http.StatusInternalServerError, "failed to get token")
		return
	}

	// Check ownership
	if token.UserID != user.ID && !user.IsAdmin {
		s.respondError(w, http.StatusForbidden, "access denied")
		return
	}

	// Delete token
	if err := s.db.Tokens.Delete(id); err != nil {
		s.log.Error().Err(err).Msg("Failed to delete token")
		s.respondError(w, http.StatusInternalServerError, "failed to delete token")
		return
	}

	// Log audit
	ipAddress := auth.GetClientIP(r)
	_ = s.db.Audit.Log(&user.ID, database.ActionTokenDeleted, map[string]interface{}{
		"token_id":   token.ID,
		"token_name": token.Name,
	}, ipAddress)

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "token deleted successfully",
	})
}
