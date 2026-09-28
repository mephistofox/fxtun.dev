package core

import "testing"

// The client reports its version with a leading "v" (release.yml bakes in
// "v3.16.1"), while min_version in server.yaml may be written either way.
// Gluing another "v" on produces "vv3.16.1", which is not a valid semver — and
// an invalid version compares LESS than every valid one, so the check rejected
// every client alive, including the newest. Written the other way round, both
// sides were invalid, compared equal, and the check never fired at all.
func TestClientVersionBelowMinimum(t *testing.T) {
	tests := []struct {
		name      string
		clientVer string
		minVer    string
		want      bool
	}{
		{"old client, min without the v", "v3.15.0", "3.16.1", true},
		{"old client, min with the v", "v3.15.0", "v3.16.1", true},
		{"current client, min without the v", "v3.16.1", "3.16.1", false},
		{"current client, min with the v", "v3.16.1", "v3.16.1", false},
		{"newer client than the minimum", "v3.17.0", "v3.16.1", false},
		{"client without the v prefix", "3.15.0", "3.16.1", true},
		{"no minimum configured", "v3.15.0", "", false},
		{"client version unknown", "", "v3.16.1", false},
		{"local dev build is never too old", "dev", "v3.16.1", false},
		{"unparseable client version is not rejected", "banana", "v3.16.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientVersionBelowMinimum(tt.clientVer, tt.minVer); got != tt.want {
				t.Fatalf("clientVersionBelowMinimum(%q, %q) = %v, want %v",
					tt.clientVer, tt.minVer, got, tt.want)
			}
		})
	}
}
