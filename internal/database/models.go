package database

import (
	"time"
)

// User represents a registered user
type User struct {
	ID           int64      `json:"id"`
	Phone        string     `json:"phone"`
	PasswordHash string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	IsAdmin      bool       `json:"is_admin"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// InviteCode represents a one-time invitation code
type InviteCode struct {
	ID              int64      `json:"id"`
	Code            string     `json:"code"`
	CreatedByUserID *int64     `json:"created_by_user_id,omitempty"`
	UsedByUserID    *int64     `json:"used_by_user_id,omitempty"`
	UsedAt          *time.Time `json:"used_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// IsUsed returns true if the invite code has been used
func (i *InviteCode) IsUsed() bool {
	return i.UsedByUserID != nil
}

// IsExpired returns true if the invite code has expired
func (i *InviteCode) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

// ReservedDomain represents a subdomain reserved by a user
type ReservedDomain struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Subdomain string    `json:"subdomain"`
	CreatedAt time.Time `json:"created_at"`
}

// Session represents a user session (refresh token)
type Session struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	RefreshTokenHash string    `json:"-"`
	UserAgent        string    `json:"user_agent,omitempty"`
	IPAddress        string    `json:"ip_address,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// IsExpired returns true if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// APIToken represents an API token for CLI authentication
type APIToken struct {
	ID                int64      `json:"id"`
	UserID            int64      `json:"user_id"`
	TokenHash         string     `json:"-"`
	Name              string     `json:"name"`
	AllowedSubdomains []string   `json:"allowed_subdomains"`
	MaxTunnels        int        `json:"max_tunnels"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// CanUseSubdomain checks if the token allows using a specific subdomain
func (t *APIToken) CanUseSubdomain(subdomain string) bool {
	for _, pattern := range t.AllowedSubdomains {
		if pattern == "*" {
			return true
		}
		if matchWildcard(pattern, subdomain) {
			return true
		}
	}
	return false
}

// matchWildcard matches a pattern like "dev-*" against a subdomain
func matchWildcard(pattern, subdomain string) bool {
	if len(pattern) == 0 {
		return len(subdomain) == 0
	}

	// Handle wildcard at the end
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(subdomain) >= len(prefix) && subdomain[:len(prefix)] == prefix
	}

	// Handle wildcard at the beginning
	if pattern[0] == '*' {
		suffix := pattern[1:]
		return len(subdomain) >= len(suffix) && subdomain[len(subdomain)-len(suffix):] == suffix
	}

	return pattern == subdomain
}

// TOTPSecret represents TOTP 2FA settings for a user
type TOTPSecret struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	SecretEncrypted string    `json:"-"`
	IsEnabled       bool      `json:"is_enabled"`
	BackupCodes     []string  `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int64                  `json:"id"`
	UserID    *int64                 `json:"user_id,omitempty"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Audit log action constants
const (
	ActionLogin           = "login"
	ActionLogout          = "logout"
	ActionRegister        = "register"
	ActionPasswordChange  = "password_change"
	ActionTokenCreated    = "token_created"
	ActionTokenDeleted    = "token_deleted"
	ActionDomainReserved  = "domain_reserved"
	ActionDomainReleased  = "domain_released"
	ActionTunnelCreated   = "tunnel_created"
	ActionTunnelClosed    = "tunnel_closed"
	ActionTOTPEnabled     = "totp_enabled"
	ActionTOTPDisabled    = "totp_disabled"
	ActionInviteCreated   = "invite_created"
	ActionInviteUsed      = "invite_used"
	ActionUserUpdated     = "user_updated"
	ActionUserDeleted     = "user_deleted"
)
