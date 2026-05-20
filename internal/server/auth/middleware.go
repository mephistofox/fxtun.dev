package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/mephistofox/fxtunnel/internal/server/database"
)

// Context keys for authentication
type contextKey string

const (
	UserContextKey         contextKey = "user"
	ClaimsContextKey       contextKey = "claims"
	OriginalRemoteAddrKey  contextKey = "originalRemoteAddr"
)

// AuthenticatedUser represents the authenticated user in context
type AuthenticatedUser struct {
	ID      int64
	Phone   string
	IsAdmin bool
	Plan    *database.Plan
}

// MiddlewareWithDB creates an authentication middleware that supports both JWT and API tokens
func MiddlewareWithDB(authService *Service, db *database.Database) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Check Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]
			var user *AuthenticatedUser

			// Check if it's an API token (sk_xxx)
			if strings.HasPrefix(token, "sk_") {
				// Hash the token and look it up
				tokenHash := HashToken(token)

				apiToken, err := db.Tokens.GetByTokenHash(tokenHash)
				if err != nil || apiToken == nil {
					http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
					return
				}

				// Get the user
				dbUser, err := db.Users.GetByID(apiToken.UserID)
				if err != nil || dbUser == nil {
					http.Error(w, `{"error": "user not found"}`, http.StatusUnauthorized)
					return
				}

				if !dbUser.IsActive {
					http.Error(w, `{"error":"user_inactive","code":"USER_INACTIVE"}`, http.StatusForbidden)
					return
				}

				var plan *database.Plan
				if dbUser.PlanID > 0 {
					plan, _ = db.Plans.GetByID(dbUser.PlanID)
				}

				user = &AuthenticatedUser{
					ID:      dbUser.ID,
					Phone:   dbUser.Phone,
					IsAdmin: dbUser.IsAdmin,
					Plan:    plan,
				}
			} else {
				// Validate as JWT
				claims, err := authService.ValidateAccessToken(token)
				if err != nil {
					if err == ErrTokenExpired {
						http.Error(w, `{"error": "token expired"}`, http.StatusUnauthorized)
						return
					}
					http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
					return
				}

				// Check if user is still active
				jwtUser, err := db.Users.GetByID(claims.UserID)
				if err != nil || jwtUser == nil {
					http.Error(w, `{"error": "user not found"}`, http.StatusUnauthorized)
					return
				}
				if !jwtUser.IsActive {
					http.Error(w, `{"error":"user_inactive","code":"USER_INACTIVE"}`, http.StatusForbidden)
					return
				}

				var plan *database.Plan
				if jwtUser.PlanID > 0 {
					plan, _ = db.Plans.GetByID(jwtUser.PlanID)
				}

				user = &AuthenticatedUser{
					ID:      jwtUser.ID,
					Phone:   jwtUser.Phone,
					IsAdmin: jwtUser.IsAdmin,
					Plan:    plan,
				}
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Middleware creates an authentication middleware (JWT only, for backwards compatibility)
func Middleware(authService *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Check Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token
			claims, err := authService.ValidateAccessToken(token)
			if err != nil {
				if err == ErrTokenExpired {
					http.Error(w, `{"error": "token expired"}`, http.StatusUnauthorized)
					return
				}
				http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Add user to context
			user := &AuthenticatedUser{
				ID:      claims.UserID,
				Phone:   claims.Phone,
				IsAdmin: claims.IsAdmin,
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			ctx = context.WithValue(ctx, ClaimsContextKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalMiddleware creates a middleware that authenticates if token is present but doesn't require it
func OptionalMiddleware(authService *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				next.ServeHTTP(w, r)
				return
			}

			token := parts[1]
			claims, err := authService.ValidateAccessToken(token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			user := &AuthenticatedUser{
				ID:      claims.UserID,
				Phone:   claims.Phone,
				IsAdmin: claims.IsAdmin,
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			ctx = context.WithValue(ctx, ClaimsContextKey, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminMiddleware creates a middleware that requires admin privileges
func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if !user.IsAdmin {
			http.Error(w, `{"error": "admin access required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the authenticated user from context
func GetUserFromContext(ctx context.Context) *AuthenticatedUser {
	user, ok := ctx.Value(UserContextKey).(*AuthenticatedUser)
	if !ok {
		return nil
	}
	return user
}

// GetClaimsFromContext retrieves the JWT claims from context
func GetClaimsFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// GetClientIP extracts the client IP address from the request.
//
// By the time a handler runs, the API server's trustedRealIPMiddleware has
// already rewritten r.RemoteAddr to the real client IP when the request
// came in through a trusted proxy (e.g. nginx on 127.0.0.1). When the
// source is untrusted, r.RemoteAddr is the raw TCP source — forwarded
// headers are ignored on purpose to prevent spoofing.
//
// This function just normalises the value by stripping the port (if any).
// Code paths that specifically need the proxy's own address (e.g. webhook
// IP allowlists where the proxy is the trust boundary) should read
// OriginalRemoteAddrKey from the request context instead.
func GetClientIP(r *http.Request) string {
	return stripPort(r.RemoteAddr)
}

// stripPort removes the port suffix from an address string.
func stripPort(addr string) string {
	if colonIdx := strings.LastIndex(addr, ":"); colonIdx != -1 {
		// Check if this is IPv6
		if strings.Contains(addr, "[") {
			// IPv6 format: [::1]:port
			if bracketIdx := strings.LastIndex(addr, "]"); bracketIdx != -1 {
				return addr[1:bracketIdx]
			}
		}
		return addr[:colonIdx]
	}
	return addr
}
