package server

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// checkBasicAuth validates HTTP Basic Auth credentials against the tunnel's stored bcrypt hash.
// Returns true if the request is authorized (either no auth required or valid credentials).
// Returns false and writes a 401 response if authentication fails.
func checkBasicAuth(w http.ResponseWriter, r *http.Request, tunnel *Tunnel) bool {
	// No auth required — backward compatible
	if tunnel.BasicAuthHash == "" {
		return true
	}

	// Extract credentials from request
	username, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="fxTunnel"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	// Compare username:password against stored bcrypt hash
	credential := username + ":" + password
	if err := bcrypt.CompareHashAndPassword([]byte(tunnel.BasicAuthHash), []byte(credential)); err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="fxTunnel"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	return true
}
