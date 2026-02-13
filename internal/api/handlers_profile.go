package api

import (
	"errors"
	"net/http"

	"github.com/mephistofox/fxtun.dev/internal/api/dto"
	"github.com/mephistofox/fxtun.dev/internal/auth"
	"github.com/mephistofox/fxtun.dev/internal/database"
)

// handleGetProfile returns the current user's profile
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get full user from database
	dbUser, err := s.db.Users.GetByID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get user")
		s.respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Get TOTP status
	totpEnabled, _ := s.db.TOTP.IsEnabled(user.ID)

	// Get reserved domains
	domains, err := s.db.Domains.GetByUserID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get domains")
		domains = []*database.ReservedDomain{}
	}

	// Get token count
	tokenCount, _ := s.db.Tokens.Count(user.ID)

	// Get active tunnel count
	tunnelCount := 0
	if s.tunnelProvider != nil {
		userTunnels := s.tunnelProvider.GetTunnelsByUserID(user.ID)
		tunnelCount = len(userTunnels)
	}

	// Load plan
	var planDTO *dto.PlanDTO
	if dbUser.PlanID > 0 {
		if plan, err := s.db.Plans.GetByID(dbUser.PlanID); err == nil {
			planDTO = dto.PlanFromModel(plan)
		}
	}

	// Convert domains to DTOs
	domainDTOs := make([]*dto.DomainDTO, len(domains))
	for i, d := range domains {
		domainDTOs[i] = dto.DomainFromModel(d, s.baseDomain)
	}

	maxDomains := 1
	if planDTO != nil {
		maxDomains = planDTO.MaxDomains
	}

	s.respondJSON(w, http.StatusOK, dto.ProfileResponse{
		User:            dto.UserFromModel(dbUser),
		TOTPEnabled:     totpEnabled,
		ReservedDomains: domainDTOs,
		MaxDomains:      maxDomains,
		TokenCount:      tokenCount,
		TunnelCount:     tunnelCount,
		Plan:            planDTO,
	})
}

// handleUpdateProfile updates the current user's profile
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.UpdateProfileRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get user from database
	dbUser, err := s.db.Users.GetByID(user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get user")
		s.respondError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Update fields
	if req.DisplayName != "" {
		dbUser.DisplayName = req.DisplayName
	}

	if err := s.db.Users.Update(dbUser); err != nil {
		s.log.Error().Err(err).Msg("Failed to update user")
		s.respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.UserFromModel(dbUser))
}

// handleChangePassword changes the current user's password
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ChangePasswordRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		s.respondError(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}

	if len(req.NewPassword) < 8 {
		s.respondError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}
	if len(req.NewPassword) > 128 {
		s.respondError(w, http.StatusBadRequest, "new password must be at most 128 characters")
		return
	}

	ipAddress := auth.GetClientIP(r)

	if err := s.authService.ChangePassword(user.ID, req.OldPassword, req.NewPassword, ipAddress); err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_PASSWORD", "current password is incorrect")
			return
		}
		s.log.Error().Err(err).Msg("Failed to change password")
		s.respondError(w, http.StatusInternalServerError, "failed to change password")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "password changed successfully",
	})
}
