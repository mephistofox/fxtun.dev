package client

import "github.com/mephistofox/fxtun.dev/internal/protocol"

// AuthError represents an authentication error with a specific code
type AuthError struct {
	Code    string
	Message string
}

func (e *AuthError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "authentication error: " + e.Code
}

// IsTokenExpired returns true if the error indicates an expired token
func (e *AuthError) IsTokenExpired() bool {
	return e.Code == protocol.ErrCodeTokenExpired
}

// NewAuthError creates a new AuthError with the given code and message
func NewAuthError(code, message string) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
	}
}
