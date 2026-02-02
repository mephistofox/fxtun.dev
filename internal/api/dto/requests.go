package dto

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Phone       string `json:"phone" validate:"required,min=10,max=20"`
	Password    string `json:"password" validate:"required,min=8,max=128"`
	InviteCode  string `json:"invite_code" validate:"required,min=16,max=32"`
	DisplayName string `json:"display_name" validate:"max=100"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Phone    string `json:"phone" validate:"required"`
	Password string `json:"password" validate:"required"`
	TOTPCode string `json:"totp_code,omitempty"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" validate:"max=100"`
}

// CreateTokenRequest represents an API token creation request
type CreateTokenRequest struct {
	Name              string   `json:"name" validate:"required,min=1,max=100"`
	AllowedSubdomains []string `json:"allowed_subdomains"`
	AllowedIPs        []string `json:"allowed_ips"`
	MaxTunnels        int      `json:"max_tunnels" validate:"min=1,max=100"`
}

// ReserveDomainRequest represents a domain reservation request
type ReserveDomainRequest struct {
	Subdomain string `json:"subdomain" validate:"required,min=3,max=32,alphanum"`
}

// CreateInviteCodeRequest represents an invite code creation request
type CreateInviteCodeRequest struct {
	ExpiresInDays int `json:"expires_in_days,omitempty"` // 0 = no expiry
}

// TOTPVerifyRequest represents a TOTP verification request
type TOTPVerifyRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

// TOTPDisableRequest represents a TOTP disable request
type TOTPDisableRequest struct {
	Code string `json:"code" validate:"required,min=6,max=8"`
}

// DeviceAuthorizeRequest represents a device flow authorization request
type DeviceAuthorizeRequest struct {
	SessionID string `json:"session_id"`
}

// UpdateUserRequest represents an admin user update request
type UpdateUserRequest struct {
	IsAdmin  *bool  `json:"is_admin,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
	PlanID   *int64 `json:"plan_id,omitempty"`
}

// CreatePlanRequest represents a plan creation request
type CreatePlanRequest struct {
	Slug               string  `json:"slug"`
	Name               string  `json:"name"`
	Price              float64 `json:"price"`
	MaxTunnels         int     `json:"max_tunnels"`
	MaxDomains         int     `json:"max_domains"`
	MaxCustomDomains   int     `json:"max_custom_domains"`
	MaxTokens          int     `json:"max_tokens"`
	MaxTunnelsPerToken int     `json:"max_tunnels_per_token"`
	InspectorEnabled   bool    `json:"inspector_enabled"`
}

// UpdatePlanRequest represents a plan update request
type UpdatePlanRequest struct {
	Name               *string  `json:"name,omitempty"`
	Price              *float64 `json:"price,omitempty"`
	MaxTunnels         *int     `json:"max_tunnels,omitempty"`
	MaxDomains         *int     `json:"max_domains,omitempty"`
	MaxCustomDomains   *int     `json:"max_custom_domains,omitempty"`
	MaxTokens          *int     `json:"max_tokens,omitempty"`
	MaxTunnelsPerToken *int     `json:"max_tunnels_per_token,omitempty"`
	InspectorEnabled   *bool    `json:"inspector_enabled,omitempty"`
}

// MergeUsersRequest represents a request to merge two users
type MergeUsersRequest struct {
	PrimaryUserID   int64 `json:"primary_user_id"`
	SecondaryUserID int64 `json:"secondary_user_id"`
}

// ResetPasswordRequest represents an admin password reset request
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}
