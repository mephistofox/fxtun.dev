package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordAndCheck(t *testing.T) {
	hash, err := HashPassword("mysecret")
	require.NoError(t, err)
	assert.True(t, CheckPassword("mysecret", hash))
}

func TestCheckPasswordWrong(t *testing.T) {
	hash, err := HashPassword("mysecret")
	require.NoError(t, err)
	assert.False(t, CheckPassword("wrong", hash))
}

func TestGenerateAPIToken(t *testing.T) {
	token, err := GenerateAPIToken()
	require.NoError(t, err)
	assert.Equal(t, "sk_fxtunnel_", token[:12])
	assert.Len(t, token, 12+48) // prefix + 24 bytes hex
}

func TestGenerateRefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken()
	require.NoError(t, err)
	assert.Equal(t, "rt_", token[:3])
	assert.Len(t, token, 3+48)
}

func TestGenerateInviteCode(t *testing.T) {
	code, err := GenerateInviteCode()
	require.NoError(t, err)
	assert.Len(t, code, 24)
}

func TestHashTokenDeterministicAndLength(t *testing.T) {
	h1 := HashToken("test")
	h2 := HashToken("test")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64)
}

func TestHashTokenDifferentInputs(t *testing.T) {
	assert.NotEqual(t, HashToken("a"), HashToken("b"))
}

func TestGenerateBackupCodes(t *testing.T) {
	codes, err := GenerateBackupCodes(5)
	require.NoError(t, err)
	assert.Len(t, codes, 5)
	unique := make(map[string]bool)
	for _, c := range codes {
		assert.Len(t, c, 8) // 4 bytes hex
		unique[c] = true
	}
	assert.Len(t, unique, 5)
}

func TestGenerateSecret(t *testing.T) {
	s, err := GenerateSecret(32)
	require.NoError(t, err)
	assert.Len(t, s, 32)
}

func TestGenerateTokenUniqueness(t *testing.T) {
	t1, err := GenerateToken("x_")
	require.NoError(t, err)
	t2, err := GenerateToken("x_")
	require.NoError(t, err)
	assert.NotEqual(t, t1, t2)
}
