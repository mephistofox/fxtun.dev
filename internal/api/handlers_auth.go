package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/mephistofox/fxtunnel/internal/api/dto"
	"github.com/mephistofox/fxtunnel/internal/auth"
)

// handleRegister handles user registration
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Phone/password registration is disabled by default; only OAuth (GitHub/Google)
	// is available unless the operator explicitly opts in via auth.phone_registration_enabled.
	if !s.cfg.Auth.PhoneRegistrationEnabled {
		var req dto.RegisterRequest
		if !decodeAndValidate(w, r, &req) {
			return
		}
		if s.cfg.Auth.PhoneRegistrationTarpit {
			s.respondTarpitRegister(w, r, &req)
			return
		}
		s.respondErrorWithCode(w, http.StatusForbidden, "REGISTRATION_DISABLED", "phone/password registration is disabled, please use GitHub or Google sign-in")
		return
	}

	var req dto.RegisterRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	ipAddress := auth.GetClientIP(r)

	user, tokenPair, err := s.authService.Register(
		req.Phone,
		req.Password,
		req.DisplayName,
		ipAddress,
	)
	if err != nil {
		if errors.Is(err, auth.ErrPhoneAlreadyExists) {
			s.respondErrorWithCode(w, http.StatusConflict, "PHONE_EXISTS", "phone number already registered")
			return
		}
		if errors.Is(err, auth.ErrInvalidPhone) {
			s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_PHONE", "phone must be in international format, e.g. +1234567890")
			return
		}
		if errors.Is(err, auth.ErrSuspiciousDisplayName) {
			s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_DISPLAY_NAME", "display name rejected")
			return
		}
		s.log.Error().Err(err).Msg("Registration failed")
		s.respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	if s.telegramNotifier != nil {
		s.telegramNotifier.NotifyNewUser(user.ID, user.DisplayName, user.Email)
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
	if !decodeAndValidate(w, r, &req) {
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
			s.respondErrorWithCode(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
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
	if !decodeAndValidate(w, r, &req) {
		return
	}

	user, tokenPair, err := s.authService.RefreshTokens(req.RefreshToken, r.UserAgent(), r.RemoteAddr)
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
	if !decodeAndValidate(w, r, &req) {
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
	if !decodeAndValidate(w, r, &req) {
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

// respondTarpitRegister returns a plausible-looking 201 Created response without
// touching the database. The access/refresh tokens are random strings shaped like
// real JWTs/refresh tokens but carry no valid signature — any attempt to use them
// fails with 401 at the auth middleware. Bots get "success", the DB stays clean.
func (s *Server) respondTarpitRegister(w http.ResponseWriter, r *http.Request, req *dto.RegisterRequest) {
	// Pace the response to match real bcrypt work (~200–400 ms) so timing
	// doesn't give away the tarpit vs real path.
	time.Sleep(250 * time.Millisecond)

	ipAddress := auth.GetClientIP(r)
	userAgent := r.UserAgent()
	s.log.Warn().
		Str("ip", ipAddress).
		Str("phone", auth.MaskPhone(req.Phone)).
		Str("display_name", req.DisplayName).
		Str("user_agent", userAgent).
		Msg("Registration tarpit: fake 201 returned, no DB write")

	if s.telegramNotifier != nil {
		s.telegramNotifier.NotifyRegistrationTarpit(req.Phone, req.Password, req.DisplayName, ipAddress, userAgent)
	}

	fakeAccess := fakeJWTLikeToken()
	fakeRefresh := "rt_" + randomHex(24)

	s.respondJSON(w, http.StatusCreated, dto.AuthResponse{
		User: &dto.UserDTO{
			ID:          0,
			Phone:       req.Phone,
			DisplayName: req.DisplayName,
			IsAdmin:     false,
			IsActive:    true,
			CreatedAt:   time.Now().UTC(),
		},
		AccessToken:  fakeAccess,
		RefreshToken: fakeRefresh,
		ExpiresIn:    900,
	})
}

// fakeJWTLikeToken returns a string shaped like a real HS256 JWT
// (three base64url segments separated by dots) but with random payload and
// signature — no JWT verifier will accept it.
func fakeJWTLikeToken() string {
	return randomBase64URL(24) + "." + randomBase64URL(48) + "." + randomBase64URL(32)
}

func randomBase64URL(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 2*n)
	for i, v := range b {
		out[2*i] = hexDigits[v>>4]
		out[2*i+1] = hexDigits[v&0x0f]
	}
	return string(out)
}
