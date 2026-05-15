package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSession_IsExpired(t *testing.T) {
	s := &Session{ExpiresAt: time.Now().Add(1 * time.Hour)}
	assert.False(t, s.IsExpired())

	s.ExpiresAt = time.Now().Add(-1 * time.Hour)
	assert.True(t, s.IsExpired())
}

func TestAPIToken_CanUseSubdomain(t *testing.T) {
	tests := []struct {
		name      string
		patterns  []string
		subdomain string
		expected  bool
	}{
		{"wildcard all", []string{"*"}, "anything", true},
		{"exact match", []string{"myapp"}, "myapp", true},
		{"exact no match", []string{"myapp"}, "other", false},
		{"prefix wildcard", []string{"dev-*"}, "dev-test", true},
		{"prefix no match", []string{"dev-*"}, "prod-test", false},
		{"suffix wildcard", []string{"*-app"}, "my-app", true},
		{"suffix no match", []string{"*-app"}, "my-svc", false},
		{"empty patterns", []string{}, "anything", false},
		{"multiple patterns", []string{"a", "b"}, "b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &APIToken{AllowedSubdomains: tt.patterns}
			assert.Equal(t, tt.expected, token.CanUseSubdomain(tt.subdomain))
		})
	}
}

func TestAPIToken_IsIPAllowed_Empty(t *testing.T) {
	token := &APIToken{AllowedIPs: []string{}}
	assert.True(t, token.IsIPAllowed("1.2.3.4:5678"))
}

func TestAPIToken_IsIPAllowed_Match(t *testing.T) {
	token := &APIToken{AllowedIPs: []string{"1.2.3.4", "5.6.7.8"}}
	assert.True(t, token.IsIPAllowed("1.2.3.4:5678"))
	assert.False(t, token.IsIPAllowed("9.9.9.9:1234"))
}

func TestAPIToken_IsIPAllowed_NoPort(t *testing.T) {
	token := &APIToken{AllowedIPs: []string{"1.2.3.4"}}
	assert.True(t, token.IsIPAllowed("1.2.3.4"))
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern   string
		subdomain string
		expected  bool
	}{
		{"", "", true},
		{"", "a", false},
		{"a", "a", true},
		{"a", "b", false},
		{"dev-*", "dev-", true},
		{"dev-*", "dev-test", true},
		{"dev-*", "prod", false},
		{"*-app", "-app", true},
		{"*-app", "my-app", true},
		{"*-app", "my-svc", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.subdomain, func(t *testing.T) {
			assert.Equal(t, tt.expected, matchWildcard(tt.pattern, tt.subdomain))
		})
	}
}
