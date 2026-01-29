package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInviteCode_IsUsed(t *testing.T) {
	inv := &InviteCode{}
	assert.False(t, inv.IsUsed())

	uid := int64(1)
	inv.UsedByUserID = &uid
	assert.True(t, inv.IsUsed())
}

func TestInviteCode_IsExpired(t *testing.T) {
	inv := &InviteCode{}
	assert.False(t, inv.IsExpired(), "nil ExpiresAt should not be expired")

	future := time.Now().Add(1 * time.Hour)
	inv.ExpiresAt = &future
	assert.False(t, inv.IsExpired(), "future time should not be expired")

	past := time.Now().Add(-1 * time.Hour)
	inv.ExpiresAt = &past
	assert.True(t, inv.IsExpired(), "past time should be expired")
}

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
