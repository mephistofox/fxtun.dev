package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTOTPManager() *TOTPManager {
	return NewTOTPManager("test-issuer", make([]byte, 32))
}

func TestTOTPGenerateSecret(t *testing.T) {
	m := newTestTOTPManager()
	secret, qr, err := m.GenerateSecret("user@test.com")
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.NotEmpty(t, qr)
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	m := newTestTOTPManager()
	original := "JBSWY3DPEHPK3PXP"
	encrypted, err := m.EncryptSecret(original)
	require.NoError(t, err)

	decrypted, err := m.DecryptSecret(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestDecryptWrongKey(t *testing.T) {
	m1 := newTestTOTPManager()
	encrypted, err := m1.EncryptSecret("secret")
	require.NoError(t, err)

	key2 := make([]byte, 32)
	key2[0] = 1
	m2 := NewTOTPManager("test", key2)
	_, err = m2.DecryptSecret(encrypted)
	assert.Error(t, err)
}

func TestDecryptInvalidBase64(t *testing.T) {
	m := newTestTOTPManager()
	_, err := m.DecryptSecret("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestDecryptTooShort(t *testing.T) {
	m := newTestTOTPManager()
	// Valid base64 but too short for nonce+ciphertext
	_, err := m.DecryptSecret("AQID")
	assert.Error(t, err)
}

func TestValidateBackupCode(t *testing.T) {
	m := newTestTOTPManager()
	codes := []string{"AAAAAAAA", "BBBBBBBB", "CCCCCCCC"}

	remaining, valid := m.ValidateBackupCode("BBBBBBBB", codes)
	assert.True(t, valid)
	assert.Len(t, remaining, 2)
	assert.NotContains(t, remaining, "BBBBBBBB")

	remaining, valid = m.ValidateBackupCode("INVALID", codes)
	assert.False(t, valid)
}

func TestTOTPGenerateBackupCodes(t *testing.T) {
	m := newTestTOTPManager()
	codes, err := m.GenerateBackupCodes(5)
	require.NoError(t, err)
	assert.Len(t, codes, 5)
	for _, c := range codes {
		assert.Len(t, c, 8)
	}
}

func TestGetQRCodeDataURL(t *testing.T) {
	url := GetQRCodeDataURL([]byte{1, 2, 3})
	assert.True(t, strings.HasPrefix(url, "data:image/png;base64,"))
}
