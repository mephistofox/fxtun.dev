package api

import (
	"errors"
	"net/http"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
)

// handleRegister handles user registration
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Phone == "" || req.Password == "" || req.InviteCode == "" {
		s.respondError(w, http.StatusBadRequest, "phone, password, and invite_code are required")
		return
	}

	if len(req.Password) < 8 {
		s.respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if len(req.Password) > 128 {
		s.respondError(w, http.StatusBadRequest, "password must be at most 128 characters")
		return
	}

	ipAddress := auth.GetClientIP(r)

	user, tokenPair, err := s.authService.Register(
		req.Phone,
		req.Password,
		req.InviteCode,
		req.DisplayName,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidInviteCode) {
			s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_INVITE", "invalid or expired invite code")
			return
		}
		if errors.Is(err, auth.ErrPhoneAlreadyExists) {
			s.respondErrorWithCode(w, http.StatusConflict, "PHONE_EXISTS", "phone number already registered")
			return
		}
		s.log.Error().Err(err).Msg("Registration failed")
		s.respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	s.respondJSON(w, http.StatusCreated, dto.AuthResponse{
		User:         dto.UserFromModel(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Phone == "" || req.Password == "" {
		s.respondError(w, http.StatusBadRequest, "phone and password are required")
		return
	}

	userAgent := r.UserAgent()
	ipAddress := auth.GetClientIP(r)

	user, tokenPair, err := s.authService.Login(
		req.Phone,
		req.Password,
		req.TOTPCode,
		userAgent,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			s.respondErrorWithCode(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid phone or password")
			return
		}
		if errors.Is(err, auth.ErrUserNotActive) {
			s.respondErrorWithCode(w, http.StatusForbidden, "USER_INACTIVE", "user account is inactive")
			return
		}
		if errors.Is(err, auth.ErrTOTPRequired) {
			s.respondErrorWithCode(w, http.StatusUnauthorized, "TOTP_REQUIRED", "TOTP code required")
			return
		}
		if errors.Is(err, auth.ErrInvalidTOTPCode) {
			s.respondErrorWithCode(w, http.StatusUnauthorized, "INVALID_TOTP", "invalid TOTP code")
			return
		}
		s.log.Error().Err(err).Msg("Login failed")
		s.respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.AuthResponse{
		User:         dto.UserFromModel(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// handleLogout handles user logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ipAddress := auth.GetClientIP(r)

	if err := s.authService.Logout(req.RefreshToken, ipAddress, user.ID); err != nil {
		s.log.Error().Err(err).Msg("Logout failed")
		// Don't return error, just log it
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "logged out successfully",
	})
}

// handleRefresh handles token refresh
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		s.respondError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	user, tokenPair, err := s.authService.RefreshTokens(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrTokenExpired) {
			s.respondErrorWithCode(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
			return
		}
		if errors.Is(err, auth.ErrUserNotActive) {
			s.respondErrorWithCode(w, http.StatusForbidden, "USER_INACTIVE", "user account is inactive")
			return
		}
		s.log.Error().Err(err).Msg("Token refresh failed")
		s.respondError(w, http.StatusInternalServerError, "token refresh failed")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.AuthResponse{
		User:         dto.UserFromModel(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// handleTOTPEnable handles TOTP enable request
func (s *Server) handleTOTPEnable(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	secret, qrCode, backupCodes, err := s.authService.EnableTOTP(user.ID, user.Phone)
	if err != nil {
		s.log.Error().Err(err).Msg("TOTP enable failed")
		s.respondError(w, http.StatusInternalServerError, "failed to enable TOTP")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.TOTPEnableResponse{
		Secret:      secret,
		QRCode:      auth.GetQRCodeDataURL(qrCode),
		BackupCodes: backupCodes,
	})
}

// handleTOTPVerify handles TOTP verification
func (s *Server) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.TOTPVerifyRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		s.respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	ipAddress := auth.GetClientIP(r)

	if err := s.authService.VerifyAndEnableTOTP(user.ID, req.Code, ipAddress); err != nil {
		if errors.Is(err, auth.ErrInvalidTOTPCode) {
			s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_CODE", "invalid TOTP code")
			return
		}
		s.log.Error().Err(err).Msg("TOTP verify failed")
		s.respondError(w, http.StatusInternalServerError, "failed to verify TOTP")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "TOTP enabled successfully",
	})
}

// handleTOTPDisable handles TOTP disable
func (s *Server) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		s.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.TOTPDisableRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		s.respondError(w, http.StatusBadRequest, "code is required")
		return
	}

	ipAddress := auth.GetClientIP(r)

	if err := s.authService.DisableTOTP(user.ID, req.Code, ipAddress); err != nil {
		if errors.Is(err, auth.ErrInvalidTOTPCode) {
			s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_CODE", "invalid TOTP or backup code")
			return
		}
		s.log.Error().Err(err).Msg("TOTP disable failed")
		s.respondError(w, http.StatusInternalServerError, "failed to disable TOTP")
		return
	}

	s.respondJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "TOTP disabled successfully",
	})
}
