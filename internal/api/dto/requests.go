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
	IsAdmin  *bool `json:"is_admin,omitempty"`
	IsActive *bool `json:"is_active,omitempty"`
}
