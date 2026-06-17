package core

import "strings"

// WebHost maps a tunnel server address to the host that serves the REST API and
// client downloads over HTTPS.
//
// The control plane may live on a dedicated host like tunnel.fxtun.dev:443 —
// a raw TLS + yamux listener that is NOT an HTTP server (it answers the first
// request byte with the compression-negotiation byte, which an HTTP client sees
// as a malformed response). The REST API and downloads are served on the base
// web host (e.g. fxtun.dev) by nginx on standard HTTPS.
//
// WebHost strips the port and a leading "tunnel." label so API/web/update calls
// target the web host rather than the control plane. Hosts without a "tunnel."
// prefix (self-hosted setups, localhost) are returned unchanged minus the port.
func WebHost(serverAddr string) string {
	host := serverAddr
	if i := strings.IndexByte(host, ':'); i != -1 {
		host = host[:i]
	}
	return strings.TrimPrefix(host, "tunnel.")
}

// WebBaseURL returns the https base URL (no port) of the REST API / web host for
// a given tunnel server address.
func WebBaseURL(serverAddr string) string {
	return "https://" + WebHost(serverAddr)
}
