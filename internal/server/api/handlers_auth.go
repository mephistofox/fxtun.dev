package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/server/api/dto"
	"github.com/mephistofox/fxtun.dev/internal/server/auth"
)

// handleRegister handles user registration
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Phone/password registration is disabled by default; only OAuth (GitHub/Google)
	// is available unless the operator explicitly opts in via auth.phone_registration_enabled.
	if !s.cfg.Auth.PhoneRegistrationEnabled {
		// If this IP was already trapped, respond with the tarpit shape immediately
		// — no body parse, no bcrypt, no Telegram spam.
		if s.cfg.Auth.PhoneRegistrationTarpit && s.ipBanStore != nil {
			clientIP := auth.GetClientIP(r)
			if banned, _, _ := s.ipBanStore.IsBanned(clientIP); banned {
				s.respondTarpitRegisterBanned(w, r)
				return
			}
		}

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
		if errors.Is(err, auth.ErrTokenReuse) {
			s.respondErrorWithCode(w, http.StatusUnauthorized, "TOKEN_REUSE", "refresh token reuse detected; all sessions revoked")
			return
		}
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

	// Ban the IP so repeat hits short-circuit without bcrypt/Telegram work.
	isNewBan := true
	var banTTL time.Duration
	if s.cfg.Auth.TarpitBanEnabled && s.ipBanStore != nil && ipAddress != "" {
		banTTL = s.cfg.Auth.TarpitBanTTL
		if banTTL <= 0 {
			banTTL = 72 * time.Hour
		}
		var err error
		isNewBan, err = s.ipBanStore.Ban(ipAddress, "registration tarpit", banTTL)
		if err != nil {
			s.log.Warn().Err(err).Str("ip", ipAddress).Msg("failed to record tarpit IP ban")
		}
	}

	if s.telegramNotifier != nil && isNewBan {
		s.telegramNotifier.NotifyRegistrationTarpit(req.Phone, req.Password, req.DisplayName, ipAddress, userAgent, banTTL)
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

// respondTarpitRegisterBanned is the fast path for IPs already trapped:
// no body parse, no bcrypt-mimicking sleep, no Telegram spam — just a plausible
// 201 with random tokens. Bots get the same shape they got the first time.
func (s *Server) respondTarpitRegisterBanned(w http.ResponseWriter, r *http.Request) {
	// Drain (bounded) body so keep-alive clients don't get RST mid-write.
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, 1<<20))
	}
	fakeAccess := fakeJWTLikeToken()
	fakeRefresh := "rt_" + randomHex(24)
	s.respondJSON(w, http.StatusCreated, dto.AuthResponse{
		User: &dto.UserDTO{
			ID:          0,
			Phone:       "",
			DisplayName: "",
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
