package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/mephistofox/fxtunnel/internal/server/api/dto"
	"github.com/mephistofox/fxtunnel/internal/server/auth"
	"github.com/mephistofox/fxtunnel/internal/server/emailguard"
	"github.com/mephistofox/fxtunnel/internal/server/store"
)

const (
	// magicLinkSendCooldown is the minimum gap between sign-in emails to the
	// same address, on top of the per-IP rate limiter.
	magicLinkSendCooldown = 60 * time.Second
	// magicLinkMaxCodeAttempts is how many wrong 6-digit codes burn the link.
	magicLinkMaxCodeAttempts = 5
	// magicLinkDefaultTTL is used when auth.magic_link_ttl is unset.
	magicLinkDefaultTTL = 20 * time.Minute
)

// magicLinkTTL returns the configured link/code validity window.
func (s *Server) magicLinkTTL() time.Duration {
	if s.cfg.Auth.MagicLinkTTL > 0 {
		return s.cfg.Auth.MagicLinkTTL
	}
	return magicLinkDefaultTTL
}

// magicLinkBaseURL returns the trusted public base URL for the emailed link,
// chosen by language. It never derives from the request Host header.
func (s *Server) magicLinkBaseURL(lang string) string {
	if lang == "en" && s.cfg.SMTP.BaseURLEN != "" {
		return strings.TrimRight(s.cfg.SMTP.BaseURLEN, "/")
	}
	if s.cfg.SMTP.BaseURL != "" {
		return strings.TrimRight(s.cfg.SMTP.BaseURL, "/")
	}
	return "https://" + s.cfg.Domain.Base
}

// handleMagicLinkSend validates the email, then (for real, deliverable,
// non-disposable addresses) emails a one-time sign-in link and 6-digit code.
//
// To avoid leaking which addresses have accounts, the response is an identical
// neutral 200 whether or not the email belongs to an existing user, and whether
// or not it was actually throttled. Only email *validity* problems (malformed,
// disposable, or undeliverable domain) return a distinct 422 so genuine users
// can fix a typo.
func (s *Server) handleMagicLinkSend(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Auth.MagicLinkEnabled {
		s.respondError(w, http.StatusNotFound, "email sign-in is not enabled")
		return
	}
	if s.notifier == nil || !s.notifier.EmailEnabled() {
		s.respondError(w, http.StatusServiceUnavailable, "email sign-in is temporarily unavailable")
		return
	}

	var req dto.MagicLinkSendRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	ipAddress := auth.GetClientIP(r)

	// Layer 1+2: syntax → disposable blocklist → MX/A deliverability check.
	normalized, err := s.emailGuard.Validate(r.Context(), req.Email)
	if err != nil {
		switch {
		case errors.Is(err, emailguard.ErrDisposableEmail):
			s.respondErrorWithCode(w, http.StatusUnprocessableEntity, "DISPOSABLE_EMAIL", "disposable email addresses are not allowed")
		case errors.Is(err, emailguard.ErrUndeliverable):
			s.respondErrorWithCode(w, http.StatusUnprocessableEntity, "UNDELIVERABLE_EMAIL", "this email domain cannot receive mail")
		default:
			s.respondErrorWithCode(w, http.StatusUnprocessableEntity, "INVALID_EMAIL", "invalid email address")
		}
		return
	}

	lang := normalizeLang(req.Lang)

	// Per-email cooldown. On a hit we still return the neutral success response
	// so callers can't probe send timing to enumerate addresses.
	if !s.magicLinkStore.AllowSend(normalized, magicLinkSendCooldown) {
		s.respondMagicLinkSent(w, lang)
		return
	}

	code, err := generateNumericCode(6)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to generate code")
		return
	}

	token, err := s.magicLinkStore.Create(&store.MagicLinkEntry{
		Email:    normalized,
		CodeHash: sha256Hex(code),
		IP:       ipAddress,
		Lang:     lang,
	}, s.magicLinkTTL())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to create sign-in link")
		return
	}

	// Build the link from a trusted, configured base URL — never from the
	// request Host header, which an attacker could spoof to have us email a
	// victim a link pointing at attacker-controlled domain with a valid token.
	link := fmt.Sprintf("%s/auth/magic?token=%s", s.magicLinkBaseURL(lang), token)
	ttlMinutes := int(s.magicLinkTTL() / time.Minute)

	// Send asynchronously so the response time is constant regardless of SMTP
	// latency (another anti-enumeration measure) and the client isn't blocked.
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error().Interface("panic", rec).Msg("Recovered from panic in magic-link email send")
			}
		}()
		if err := s.notifier.SendMagicLink(normalized, lang, link, code, ttlMinutes); err != nil {
			s.log.Error().Err(err).Msg("Failed to send magic-link email")
		}
	}()

	s.log.Info().Str("ip", ipAddress).Msg("Magic-link requested")
	s.respondMagicLinkSent(w, lang)
}

// respondMagicLinkSent writes the neutral "we sent it if valid" response.
func (s *Server) respondMagicLinkSent(w http.ResponseWriter, lang string) {
	msg := "Если адрес указан верно, мы отправили на него ссылку для входа."
	if lang == "en" {
		msg = "If the address is valid, we've sent a sign-in link to it."
	}
	s.respondJSON(w, http.StatusOK, map[string]string{"status": "sent", "message": msg})
}

// handleMagicLinkVerify completes a passwordless sign-in via either the link
// token or the email + 6-digit code, then logs the user in (creating the
// account on first sign-in).
func (s *Server) handleMagicLinkVerify(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Auth.MagicLinkEnabled {
		s.respondError(w, http.StatusNotFound, "email sign-in is not enabled")
		return
	}

	var req dto.MagicLinkVerifyRequest
	if !decodeAndValidate(w, r, &req) {
		return
	}

	// Resolve the pending entry WITHOUT consuming it yet, so a TOTP challenge
	// (below) doesn't burn the one-time link on the first round-trip. The link
	// is consumed only once sign-in fully succeeds.
	var entry *store.MagicLinkEntry
	var consumeToken string
	switch {
	case req.Token != "":
		if e, ok := s.magicLinkStore.LookupByToken(req.Token); ok {
			entry = e
			consumeToken = req.Token
		}
	case req.Email != "" && req.Code != "":
		normalized, _, nerr := emailguard.Normalize(req.Email)
		if nerr != nil {
			s.respondErrorWithCode(w, http.StatusBadRequest, "INVALID_EMAIL", "invalid email address")
			return
		}
		token, pending, ok := s.magicLinkStore.LookupByEmail(normalized)
		if ok {
			if subtle.ConstantTimeCompare([]byte(sha256Hex(req.Code)), []byte(pending.CodeHash)) == 1 {
				entry = pending
				consumeToken = token
			} else if s.magicLinkStore.IncrAttempt(normalized, s.magicLinkTTL()) >= magicLinkMaxCodeAttempts {
				// Too many wrong codes — invalidate the link entirely.
				s.magicLinkStore.ConsumeToken(token)
			}
		}
	default:
		s.respondErrorWithCode(w, http.StatusBadRequest, "MISSING_CREDENTIALS", "token or email and code are required")
		return
	}

	if entry == nil {
		s.respondErrorWithCode(w, http.StatusUnauthorized, "INVALID_LINK", "this sign-in link or code is invalid or has expired")
		return
	}

	userAgent := r.UserAgent()
	ipAddress := auth.GetClientIP(r)

	user, tokenPair, isNew, err := s.authService.RegisterOrLoginByEmail(entry.Email, "", req.TOTPCode, userAgent, ipAddress)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrTOTPRequired):
			// 2FA account — ask for the code; leave the link valid for the retry.
			s.respondErrorWithCode(w, http.StatusUnauthorized, "TOTP_REQUIRED", "TOTP code required")
			return
		case errors.Is(err, auth.ErrInvalidTOTPCode):
			s.respondErrorWithCode(w, http.StatusUnauthorized, "INVALID_TOTP", "invalid TOTP code")
			return
		case errors.Is(err, auth.ErrUserNotActive):
			s.respondErrorWithCode(w, http.StatusForbidden, "USER_INACTIVE", "user account is inactive")
			return
		}
		s.log.Error().Err(err).Msg("Magic-link login failed")
		s.respondError(w, http.StatusInternalServerError, "sign-in failed")
		return
	}

	// Sign-in succeeded — burn the one-time link now.
	s.magicLinkStore.ConsumeToken(consumeToken)

	if isNew && s.telegramNotifier != nil {
		s.telegramNotifier.NotifyNewUser(user.ID, user.DisplayName, user.Email)
	}

	status := http.StatusOK
	if isNew {
		status = http.StatusCreated
	}
	s.respondJSON(w, status, dto.AuthResponse{
		User:         dto.UserFromModel(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// normalizeLang clamps the requested language to a supported value.
func normalizeLang(lang string) string {
	if lang == "en" {
		return "en"
	}
	return "ru"
}

// sha256Hex returns the hex-encoded SHA-256 of s.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// generateNumericCode returns a cryptographically random decimal string of n digits.
func generateNumericCode(n int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		b[i] = digits[idx.Int64()]
	}
	return string(b), nil
}
