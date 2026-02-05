package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mephistofox/fxtunnel/internal/database"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActive      = errors.New("user account is not active")
	ErrPhoneAlreadyExists = errors.New("phone number already registered")
	ErrTOTPRequired       = errors.New("TOTP code required")
)

// Service handles authentication operations
type Service struct {
	db          *database.Database
	jwt         *JWTManager
	totp        *TOTPManager
	log         zerolog.Logger
	maxDomains  int
}

// NewService creates a new auth service
func NewService(db *database.Database, jwtSecret string, accessTTL, refreshTTL time.Duration, totpIssuer string, totpKey []byte, maxDomains int, log zerolog.Logger) *Service {
	return &Service{
		db:         db,
		jwt:        NewJWTManager(jwtSecret, accessTTL, refreshTTL),
		totp:       NewTOTPManager(totpIssuer, totpKey),
		log:        log.With().Str("component", "auth").Logger(),
		maxDomains: maxDomains,
	}
}

// Register creates a new user account
func (s *Service) Register(phone, password, displayName, ipAddress string) (*database.User, *TokenPair, error) {
	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	// Assign default plan
	var defaultPlanID int64
	if defaultPlan, err := s.db.Plans.GetDefault(); err == nil {
		defaultPlanID = defaultPlan.ID
	}

	// Create user
	user := &database.User{
		Phone:        phone,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		IsActive:     true,
		IsAdmin:      false,
		PlanID:       defaultPlanID,
	}

	if err := s.db.Users.Create(user); err != nil {
		if errors.Is(err, database.ErrUserAlreadyExists) {
			return nil, nil, ErrPhoneAlreadyExists
		}
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Generate tokens
	tokenPair, refreshTokenHash, err := s.jwt.GenerateTokenPair(user.ID, user.Phone, user.IsAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Create session
	session := &database.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        time.Now().Add(s.jwt.GetRefreshTokenTTL()),
	}
	if err := s.db.Sessions.Create(session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	// Log audit
	_ = s.db.Audit.Log(&user.ID, database.ActionRegister, map[string]interface{}{
		"phone": phone,
	}, ipAddress)

	s.log.Info().Int64("user_id", user.ID).Str("phone", phone).Msg("User registered")

	return user, tokenPair, nil
}

// Login authenticates a user and returns tokens.
// The identifier can be a phone number or email address.
func (s *Service) Login(identifier, password, totpCode, userAgent, ipAddress string) (*database.User, *TokenPair, error) {
	var user *database.User
	var err error

	// Try email first if it contains @, otherwise treat as phone
	if strings.Contains(identifier, "@") {
		user, err = s.db.Users.GetByEmail(identifier)
	} else {
		identifier = normalizePhone(identifier)
		user, err = s.db.Users.GetByPhone(identifier)
	}
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, nil, ErrUserNotActive
	}

	// Check password
	if !CheckPassword(password, user.PasswordHash) {
		return nil, nil, ErrInvalidCredentials
	}

	// Check TOTP if enabled
	totpEnabled, err := s.db.TOTP.IsEnabled(user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("check TOTP status: %w", err)
	}

	if totpEnabled {
		if totpCode == "" {
			return nil, nil, ErrTOTPRequired
		}

		totpSecret, err := s.db.TOTP.GetByUserID(user.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("get TOTP secret: %w", err)
		}

		// Decrypt secret
		secret, err := s.totp.DecryptSecret(totpSecret.SecretEncrypted)
		if err != nil {
			return nil, nil, fmt.Errorf("decrypt TOTP secret: %w", err)
		}

		// Validate code
		if !s.totp.ValidateCode(secret, totpCode) {
			// Try backup codes
			remainingCodes, valid := s.totp.ValidateBackupCode(totpCode, totpSecret.BackupCodes)
			if !valid {
				return nil, nil, ErrInvalidTOTPCode
			}
			// Update remaining backup codes
			if err := s.db.TOTP.UpdateBackupCodes(user.ID, remainingCodes); err != nil {
				s.log.Error().Err(err).Int64("user_id", user.ID).Msg("Failed to update backup codes")
			}
		}
	}

	// Generate tokens
	tokenPair, refreshTokenHash, err := s.jwt.GenerateTokenPair(user.ID, userIdentifier(user), user.IsAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Create session
	session := &database.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        userAgent,
		IPAddress:        ipAddress,
		ExpiresAt:        time.Now().Add(s.jwt.GetRefreshTokenTTL()),
	}
	if err := s.db.Sessions.Create(session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	// Update last login
	_ = s.db.Users.UpdateLastLogin(user.ID)

	// Log audit
	_ = s.db.Audit.Log(&user.ID, database.ActionLogin, map[string]interface{}{
		"user_agent": userAgent,
	}, ipAddress)

	s.log.Info().Int64("user_id", user.ID).Str("identifier", identifier).Msg("User logged in")

	return user, tokenPair, nil
}

// Logout invalidates a refresh token
func (s *Service) Logout(refreshToken string, ipAddress string, userID int64) error {
	tokenHash := HashToken(refreshToken)
	if err := s.db.Sessions.DeleteByTokenHash(tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	// Log audit
	_ = s.db.Audit.Log(&userID, database.ActionLogout, nil, ipAddress)

	return nil
}

// RefreshTokens generates new tokens using a refresh token
func (s *Service) RefreshTokens(refreshToken string) (*database.User, *TokenPair, error) {
	tokenHash := HashToken(refreshToken)

	// Get session
	session, err := s.db.Sessions.GetByTokenHash(tokenHash)
	if err != nil {
		if errors.Is(err, database.ErrSessionNotFound) {
			return nil, nil, ErrInvalidToken
		}
		return nil, nil, fmt.Errorf("get session: %w", err)
	}

	// Check if session is expired
	if session.IsExpired() {
		_ = s.db.Sessions.Delete(session.ID)
		return nil, nil, ErrTokenExpired
	}

	// Get user
	user, err := s.db.Users.GetByID(session.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, nil, ErrUserNotActive
	}

	// Delete old session
	_ = s.db.Sessions.Delete(session.ID)

	// Generate new tokens
	tokenPair, newRefreshTokenHash, err := s.jwt.GenerateTokenPair(user.ID, userIdentifier(user), user.IsAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Create new session
	newSession := &database.Session{
		UserID:           user.ID,
		RefreshTokenHash: newRefreshTokenHash,
		UserAgent:        session.UserAgent,
		IPAddress:        session.IPAddress,
		ExpiresAt:        time.Now().Add(s.jwt.GetRefreshTokenTTL()),
	}
	if err := s.db.Sessions.Create(newSession); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return user, tokenPair, nil
}

// ValidateAccessToken validates an access token and returns claims
func (s *Service) ValidateAccessToken(token string) (*Claims, error) {
	return s.jwt.ValidateAccessToken(token)
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(userID int64, oldPassword, newPassword, ipAddress string) error {
	user, err := s.db.Users.GetByID(userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// Verify old password
	if !CheckPassword(oldPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	// Hash new password
	newPasswordHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update password
	if err := s.db.Users.UpdatePassword(userID, newPasswordHash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Invalidate all sessions
	_ = s.db.Sessions.DeleteByUserID(userID)

	// Log audit
	_ = s.db.Audit.Log(&userID, database.ActionPasswordChange, nil, ipAddress)

	s.log.Info().Int64("user_id", userID).Msg("Password changed")

	return nil
}

// EnableTOTP enables TOTP for a user
func (s *Service) EnableTOTP(userID int64, phone string) (secret string, qrCode []byte, backupCodes []string, err error) {
	// Generate TOTP secret
	secret, qrCode, err = s.totp.GenerateSecret(phone)
	if err != nil {
		return "", nil, nil, fmt.Errorf("generate TOTP secret: %w", err)
	}

	// Encrypt secret
	encryptedSecret, err := s.totp.EncryptSecret(secret)
	if err != nil {
		return "", nil, nil, fmt.Errorf("encrypt TOTP secret: %w", err)
	}

	// Generate backup codes
	backupCodes, err = s.totp.GenerateBackupCodes(10)
	if err != nil {
		return "", nil, nil, fmt.Errorf("generate backup codes: %w", err)
	}

	// Check if TOTP already exists
	existing, err := s.db.TOTP.GetByUserID(userID)
	if err == nil && existing != nil {
		// Update existing
		existing.SecretEncrypted = encryptedSecret
		existing.BackupCodes = backupCodes
		existing.IsEnabled = false // Will be enabled after verification
		if err := s.db.TOTP.Update(existing); err != nil {
			return "", nil, nil, fmt.Errorf("update TOTP secret: %w", err)
		}
	} else {
		// Create new
		totpSecret := &database.TOTPSecret{
			UserID:          userID,
			SecretEncrypted: encryptedSecret,
			IsEnabled:       false,
			BackupCodes:     backupCodes,
		}
		if err := s.db.TOTP.Create(totpSecret); err != nil {
			return "", nil, nil, fmt.Errorf("create TOTP secret: %w", err)
		}
	}

	return secret, qrCode, backupCodes, nil
}

// VerifyAndEnableTOTP verifies a TOTP code and enables 2FA
func (s *Service) VerifyAndEnableTOTP(userID int64, code, ipAddress string) error {
	totpSecret, err := s.db.TOTP.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("get TOTP secret: %w", err)
	}

	// Decrypt secret
	secret, err := s.totp.DecryptSecret(totpSecret.SecretEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt TOTP secret: %w", err)
	}

	// Validate code
	if !s.totp.ValidateCode(secret, code) {
		return ErrInvalidTOTPCode
	}

	// Enable TOTP
	if err := s.db.TOTP.Enable(userID); err != nil {
		return fmt.Errorf("enable TOTP: %w", err)
	}

	// Log audit
	_ = s.db.Audit.Log(&userID, database.ActionTOTPEnabled, nil, ipAddress)

	s.log.Info().Int64("user_id", userID).Msg("TOTP enabled")

	return nil
}

// DisableTOTP disables TOTP for a user
func (s *Service) DisableTOTP(userID int64, code, ipAddress string) error {
	totpSecret, err := s.db.TOTP.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("get TOTP secret: %w", err)
	}

	// Decrypt secret
	secret, err := s.totp.DecryptSecret(totpSecret.SecretEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt TOTP secret: %w", err)
	}

	// Validate code
	if !s.totp.ValidateCode(secret, code) {
		// Try backup codes
		_, valid := s.totp.ValidateBackupCode(code, totpSecret.BackupCodes)
		if !valid {
			return ErrInvalidTOTPCode
		}
	}

	// Delete TOTP secret
	if err := s.db.TOTP.Delete(userID); err != nil {
		return fmt.Errorf("delete TOTP secret: %w", err)
	}

	// Log audit
	_ = s.db.Audit.Log(&userID, database.ActionTOTPDisabled, nil, ipAddress)

	s.log.Info().Int64("user_id", userID).Msg("TOTP disabled")

	return nil
}

// IsTOTPEnabled checks if TOTP is enabled for a user
func (s *Service) IsTOTPEnabled(userID int64) (bool, error) {
	return s.db.TOTP.IsEnabled(userID)
}

// OAuthUserInfo contains user information from an OAuth provider
type OAuthUserInfo struct {
	GitHubID    int64
	Email       string
	DisplayName string
	AvatarURL   string
}

// RegisterOrLoginOAuth authenticates a user via OAuth, creating the account if needed
func (s *Service) RegisterOrLoginOAuth(info *OAuthUserInfo, userAgent, ipAddress string) (*database.User, *TokenPair, error) {
	// Try to find existing user by GitHub ID
	user, err := s.db.Users.GetByGitHubID(info.GitHubID)
	if err != nil && !errors.Is(err, database.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("get user by github id: %w", err)
	}

	if user == nil {
		// Create new OAuth user
		var oauthPlanID int64
		if dp, err := s.db.Plans.GetDefault(); err == nil {
			oauthPlanID = dp.ID
		}
		user = &database.User{
			DisplayName: info.DisplayName,
			IsActive:    true,
			IsAdmin:     false,
			GitHubID:    &info.GitHubID,
			Email:       info.Email,
			AvatarURL:   info.AvatarURL,
			PlanID:      oauthPlanID,
		}
		if err := s.db.Users.CreateOAuth(user); err != nil {
			return nil, nil, fmt.Errorf("create oauth user: %w", err)
		}

		_ = s.db.Audit.Log(&user.ID, database.ActionRegister, map[string]interface{}{
			"method":    "github",
			"github_id": info.GitHubID,
		}, ipAddress)

		s.log.Info().Int64("user_id", user.ID).Int64("github_id", info.GitHubID).Msg("OAuth user registered")
	}

	if !user.IsActive {
		return nil, nil, ErrUserNotActive
	}

	// Update email from OAuth if user has no email
	if user.Email == "" && info.Email != "" {
		_ = s.db.Users.UpdateEmail(user.ID, info.Email)
		user.Email = info.Email
	}

	// Set phone to email if phone is empty (OAuth users have no phone)
	if user.Phone == "" && info.Email != "" {
		_ = s.db.Users.UpdatePhone(user.ID, info.Email)
		user.Phone = info.Email
	}

	// Generate tokens
	tokenPair, refreshTokenHash, err := s.jwt.GenerateTokenPair(user.ID, userIdentifier(user), user.IsAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Create session
	session := &database.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        userAgent,
		IPAddress:        ipAddress,
		ExpiresAt:        time.Now().Add(s.jwt.GetRefreshTokenTTL()),
	}
	if err := s.db.Sessions.Create(session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	// Update last login
	_ = s.db.Users.UpdateLastLogin(user.ID)

	_ = s.db.Audit.Log(&user.ID, database.ActionLogin, map[string]interface{}{
		"method":     "github",
		"user_agent": userAgent,
	}, ipAddress)

	s.log.Info().Int64("user_id", user.ID).Int64("github_id", info.GitHubID).Msg("OAuth user logged in")

	return user, tokenPair, nil
}

// LinkGitHub links a GitHub account to an existing user
func (s *Service) LinkGitHub(userID, githubID int64, email, avatarURL string) error {
	return s.db.Users.LinkGitHub(userID, githubID, email, avatarURL)
}

// GoogleOAuthUserInfo contains user information from Google OAuth
type GoogleOAuthUserInfo struct {
	GoogleID    string
	Email       string
	DisplayName string
	AvatarURL   string
}

// RegisterOrLoginGoogleOAuth authenticates a user via Google OAuth, creating the account if needed
func (s *Service) RegisterOrLoginGoogleOAuth(info *GoogleOAuthUserInfo, userAgent, ipAddress string) (*database.User, *TokenPair, error) {
	// Try to find existing user by Google ID
	user, err := s.db.Users.GetByGoogleID(info.GoogleID)
	if err != nil && !errors.Is(err, database.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("get user by google id: %w", err)
	}

	if user == nil {
		// Create new OAuth user
		var googlePlanID int64
		if dp, err := s.db.Plans.GetDefault(); err == nil {
			googlePlanID = dp.ID
		}
		user = &database.User{
			DisplayName: info.DisplayName,
			IsActive:    true,
			IsAdmin:     false,
			GoogleID:    &info.GoogleID,
			Email:       info.Email,
			AvatarURL:   info.AvatarURL,
			PlanID:      googlePlanID,
		}
		if err := s.db.Users.CreateOAuth(user); err != nil {
			return nil, nil, fmt.Errorf("create oauth user: %w", err)
		}

		_ = s.db.Audit.Log(&user.ID, database.ActionRegister, map[string]interface{}{
			"method":    "google",
			"google_id": info.GoogleID,
		}, ipAddress)

		s.log.Info().Int64("user_id", user.ID).Str("google_id", info.GoogleID).Msg("Google OAuth user registered")
	}

	if !user.IsActive {
		return nil, nil, ErrUserNotActive
	}

	// Update email from OAuth if user has no email
	if user.Email == "" && info.Email != "" {
		_ = s.db.Users.UpdateEmail(user.ID, info.Email)
		user.Email = info.Email
	}

	// Set phone to email if phone is empty (OAuth users have no phone)
	if user.Phone == "" && info.Email != "" {
		_ = s.db.Users.UpdatePhone(user.ID, info.Email)
		user.Phone = info.Email
	}

	// Generate tokens
	tokenPair, refreshTokenHash, err := s.jwt.GenerateTokenPair(user.ID, userIdentifier(user), user.IsAdmin)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Create session
	session := &database.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        userAgent,
		IPAddress:        ipAddress,
		ExpiresAt:        time.Now().Add(s.jwt.GetRefreshTokenTTL()),
	}
	if err := s.db.Sessions.Create(session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	// Update last login
	_ = s.db.Users.UpdateLastLogin(user.ID)

	_ = s.db.Audit.Log(&user.ID, database.ActionLogin, map[string]interface{}{
		"method":     "google",
		"user_agent": userAgent,
	}, ipAddress)

	s.log.Info().Int64("user_id", user.ID).Str("google_id", info.GoogleID).Msg("Google OAuth user logged in")

	return user, tokenPair, nil
}

// LinkGoogle links a Google account to an existing user
func (s *Service) LinkGoogle(userID int64, googleID, email, avatarURL string) error {
	return s.db.Users.LinkGoogle(userID, googleID, email, avatarURL)
}

// GetMaxDomains returns the maximum number of domains per user
func (s *Service) GetMaxDomains() int {
	return s.maxDomains
}

// GetJWTManager returns the JWT manager
func (s *Service) GetJWTManager() *JWTManager {
	return s.jwt
}

// userIdentifier returns the best identifier for a user (email if phone is empty)
func userIdentifier(user *database.User) string {
	if user.Phone != "" {
		return user.Phone
	}
	return user.Email
}

// normalizePhone removes all non-digit characters except leading +
func normalizePhone(phone string) string {
	if len(phone) == 0 {
		return phone
	}

	result := make([]byte, 0, len(phone))
	for i := 0; i < len(phone); i++ {
		c := phone[i]
		if c >= '0' && c <= '9' {
			result = append(result, c)
		} else if c == '+' && len(result) == 0 {
			result = append(result, c)
		}
	}
	return string(result)
}
