package core

import "testing"

// TestCustomDomainOwnerMismatch documents the routing rule that closes the
// released-subdomain exposure: a request arriving via a verified custom domain
// (customOwnerID >= 0) may only be routed to a tunnel owned by the same user.
// A negative customOwnerID means the request did not arrive via a custom
// domain, so there is no constraint.
func TestCustomDomainOwnerMismatch(t *testing.T) {
	cases := []struct {
		name          string
		customOwnerID int64
		tunnelOwnerID int64
		wantMismatch  bool
	}{
		{"no custom domain (sentinel)", -1, 42, false},
		{"custom domain, same owner", 7, 7, false},
		{"custom domain, different owner (released+retaken)", 7, 9, true},
		{"custom domain owner zero matches anon tunnel", 0, 0, false},
		{"custom domain owner mismatch with anon tunnel", 7, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := customDomainOwnerMismatch(tc.customOwnerID, tc.tunnelOwnerID); got != tc.wantMismatch {
				t.Errorf("customDomainOwnerMismatch(%d, %d) = %v, want %v",
					tc.customOwnerID, tc.tunnelOwnerID, got, tc.wantMismatch)
			}
		})
	}
}
