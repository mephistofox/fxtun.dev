package tls

import (
	"testing"
)

func TestValidateCustomDomain(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		baseDomain string
		wantErr    bool
		errContain string
	}{
		{
			name:       "valid domain",
			domain:     "app.example.com",
			baseDomain: "tunnel.dev",
			wantErr:    false,
		},
		{
			name:       "empty domain",
			domain:     "",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "domain is required",
		},
		{
			name:       "no dot in domain",
			domain:     "localhost-thing",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "invalid domain format",
		},
		{
			name:       "IP address rejected",
			domain:     "192.168.1.1",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "IP addresses are not allowed",
		},
		{
			name:       "IPv6 address rejected",
			domain:     "::1",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "invalid domain format",
		},
		{
			name:       "exact base domain rejected",
			domain:     "tunnel.dev",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "cannot use base domain",
		},
		{
			name:       "base domain case insensitive",
			domain:     "TUNNEL.DEV",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "cannot use base domain",
		},
		{
			name:       "subdomain of base domain rejected",
			domain:     "foo.tunnel.dev",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "cannot use base domain",
		},
		{
			name:       "deep subdomain of base domain rejected",
			domain:     "a.b.tunnel.dev",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "cannot use base domain",
		},
		{
			name:       "localhost rejected",
			domain:     "localhost",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "invalid domain format",
		},
		{
			name:       "subdomain of localhost rejected",
			domain:     "foo.localhost",
			baseDomain: "tunnel.dev",
			wantErr:    true,
			errContain: "localhost is not allowed",
		},
		{
			name:       "similar but different domain allowed",
			domain:     "mytunnel.dev",
			baseDomain: "tunnel.dev",
			wantErr:    false,
		},
		{
			name:       "completely different domain allowed",
			domain:     "my-app.example.org",
			baseDomain: "tunnel.dev",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomDomain(tt.domain, tt.baseDomain)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContain)
				}
				if tt.errContain != "" && !containsStr(err.Error(), tt.errContain) {
					t.Fatalf("expected error containing %q, got %q", tt.errContain, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
