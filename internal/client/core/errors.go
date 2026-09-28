package core

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

// IsTokenRejected reports a token the server refused outright. Every such
// server path answers "invalid token"; other failures (a hub outage says
// "authentication failed" with the same code) may pass on a retry.
func (e *AuthError) IsTokenRejected() bool {
	return e.Message == "invalid token"
}

// NewAuthError creates a new AuthError with the given code and message
func NewAuthError(code, message string) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
	}
}
