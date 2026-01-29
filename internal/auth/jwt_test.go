package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTRoundTrip(t *testing.T) {
	m := NewJWTManager("secret", 5*time.Minute, 24*time.Hour)
	token, err := m.GenerateAccessToken(42, "+1234", true)
	require.NoError(t, err)

	claims, err := m.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "+1234", claims.Phone)
	assert.True(t, claims.IsAdmin)
}

func TestJWTExpired(t *testing.T) {
	m := NewJWTManager("secret", time.Millisecond, time.Hour)
	token, err := m.GenerateAccessToken(1, "phone", false)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = m.ValidateAccessToken(token)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestJWTInvalidString(t *testing.T) {
	m := NewJWTManager("secret", time.Hour, time.Hour)
	_, err := m.ValidateAccessToken("not.a.token")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTWrongSecret(t *testing.T) {
	m1 := NewJWTManager("secret1", time.Hour, time.Hour)
	m2 := NewJWTManager("secret2", time.Hour, time.Hour)
	token, err := m1.GenerateAccessToken(1, "p", false)
	require.NoError(t, err)

	_, err = m2.ValidateAccessToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestGenerateTokenPair(t *testing.T) {
	m := NewJWTManager("secret", 5*time.Minute, 24*time.Hour)
	pair, refreshHash, err := m.GenerateTokenPair(1, "phone", false)
	require.NoError(t, err)

	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.NotEmpty(t, refreshHash)

	// Access token should be valid
	_, err = m.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)

	// Refresh hash should match
	assert.Equal(t, HashToken(pair.RefreshToken), refreshHash)
}

func TestTTLGetters(t *testing.T) {
	m := NewJWTManager("s", 5*time.Minute, 24*time.Hour)
	assert.Equal(t, 5*time.Minute, m.GetAccessTokenTTL())
	assert.Equal(t, 24*time.Hour, m.GetRefreshTokenTTL())
}
