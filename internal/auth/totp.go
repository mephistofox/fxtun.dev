package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"io"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	ErrInvalidTOTPCode = errors.New("invalid TOTP code")
	ErrTOTPNotEnabled  = errors.New("TOTP not enabled")
)

// TOTPManager handles TOTP operations
type TOTPManager struct {
	issuer        string
	encryptionKey []byte // 32 bytes for AES-256
}

// NewTOTPManager creates a new TOTP manager
func NewTOTPManager(issuer string, encryptionKey []byte) *TOTPManager {
	return &TOTPManager{
		issuer:        issuer,
		encryptionKey: encryptionKey,
	}
}

// GenerateSecret generates a new TOTP secret for a user
func (m *TOTPManager) GenerateSecret(accountName string) (secret string, qrCode []byte, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      m.issuer,
		AccountName: accountName,
		Period:      30,
		SecretSize:  20,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", nil, fmt.Errorf("generate TOTP key: %w", err)
	}

	// Generate QR code image
	img, err := key.Image(200, 200)
	if err != nil {
		return "", nil, fmt.Errorf("generate QR code image: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", nil, fmt.Errorf("encode QR code PNG: %w", err)
	}

	return key.Secret(), buf.Bytes(), nil
}

// ValidateCode validates a TOTP code against a secret
func (m *TOTPManager) ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// EncryptSecret encrypts a TOTP secret for storage
func (m *TOTPManager) EncryptSecret(secret string) (string, error) {
	plaintext := []byte(secret)

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret decrypts a stored TOTP secret
func (m *TOTPManager) DecryptSecret(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateBackupCodes generates backup codes for TOTP recovery
func (m *TOTPManager) GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 5)
		if _, err := rand.Read(bytes); err != nil {
			return nil, fmt.Errorf("generate random bytes: %w", err)
		}
		codes[i] = base32.StdEncoding.EncodeToString(bytes)[:8]
	}
	return codes, nil
}

// HashBackupCode returns a SHA-256 hex hash of a backup code with a "sha256:" prefix.
func HashBackupCode(code string) string {
	h := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return "sha256:" + hex.EncodeToString(h[:])
}

// HashBackupCodes hashes all backup codes for secure storage.
func HashBackupCodes(codes []string) []string {
	hashed := make([]string, len(codes))
	for i, c := range codes {
		hashed[i] = HashBackupCode(c)
	}
	return hashed
}

// ValidateBackupCode validates a backup code against stored codes using constant-time comparison.
// Supports both hashed (sha256:...) and legacy plaintext codes.
func (m *TOTPManager) ValidateBackupCode(code string, validCodes []string) (remaining []string, valid bool) {
	codeHash := HashBackupCode(code)
	codeUpper := strings.ToUpper(strings.TrimSpace(code))

	matchIdx := -1
	for i, stored := range validCodes {
		if strings.HasPrefix(stored, "sha256:") {
			// Hashed code — constant-time comparison
			if subtle.ConstantTimeCompare([]byte(stored), []byte(codeHash)) == 1 {
				matchIdx = i
			}
		} else {
			// Legacy plaintext code — constant-time comparison
			storedUpper := strings.ToUpper(strings.TrimSpace(stored))
			if subtle.ConstantTimeCompare([]byte(storedUpper), []byte(codeUpper)) == 1 {
				matchIdx = i
			}
		}
	}

	if matchIdx == -1 {
		return validCodes, false
	}

	remaining = make([]string, 0, len(validCodes)-1)
	remaining = append(remaining, validCodes[:matchIdx]...)
	remaining = append(remaining, validCodes[matchIdx+1:]...)
	return remaining, true
}

// GetQRCodeDataURL returns the QR code as a data URL for embedding in HTML
func GetQRCodeDataURL(qrCode []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrCode)
}
