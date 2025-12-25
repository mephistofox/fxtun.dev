// Package keyring provides secure credential storage using system keyrings.
// On macOS, it uses Keychain.
// On Linux, it uses Secret Service (GNOME Keyring, KWallet).
// On Windows, it uses Credential Manager.
package keyring

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the name used to identify fxTunnel credentials in the system keyring
	ServiceName = "fxtunnel"

	// Key constants for stored credentials
	KeyAuthToken    = "auth_token"
	KeyJWT          = "jwt"
	KeyRefreshToken = "refresh_token"
	KeyServerAddr   = "server_address"
	KeyAuthMethod   = "auth_method"
	KeyPhone        = "phone"
)

var (
	// ErrNotFound is returned when a key is not found in the keyring
	ErrNotFound = errors.New("keyring: key not found")
	// ErrAccessDenied is returned when access to the keyring is denied
	ErrAccessDenied = errors.New("keyring: access denied")
)

// Keyring provides secure credential storage
type Keyring struct {
	service string
}

// New creates a new Keyring instance
func New() *Keyring {
	return &Keyring{
		service: ServiceName,
	}
}

// NewWithService creates a new Keyring instance with a custom service name
func NewWithService(service string) *Keyring {
	return &Keyring{
		service: service,
	}
}

// Set stores a value in the keyring
func (k *Keyring) Set(key, value string) error {
	err := keyring.Set(k.service, key, value)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// Get retrieves a value from the keyring
func (k *Keyring) Get(key string) (string, error) {
	value, err := keyring.Get(k.service, key)
	if err != nil {
		return "", wrapError(err)
	}
	return value, nil
}

// Delete removes a value from the keyring
func (k *Keyring) Delete(key string) error {
	err := keyring.Delete(k.service, key)
	if err != nil {
		return wrapError(err)
	}
	return nil
}

// Exists checks if a key exists in the keyring
func (k *Keyring) Exists(key string) bool {
	_, err := k.Get(key)
	return err == nil
}

// Clear removes all fxTunnel credentials from the keyring
func (k *Keyring) Clear() error {
	keys := []string{
		KeyAuthToken,
		KeyJWT,
		KeyRefreshToken,
		KeyServerAddr,
		KeyAuthMethod,
		KeyPhone,
	}

	var lastErr error
	for _, key := range keys {
		if err := k.Delete(key); err != nil && !errors.Is(err, ErrNotFound) {
			lastErr = err
		}
	}
	return lastErr
}

// SaveCredentials saves all authentication credentials
func (k *Keyring) SaveCredentials(creds Credentials) error {
	if creds.ServerAddress != "" {
		if err := k.Set(KeyServerAddr, creds.ServerAddress); err != nil {
			return err
		}
	}

	if err := k.Set(KeyAuthMethod, creds.AuthMethod); err != nil {
		return err
	}

	if creds.Token != "" {
		if err := k.Set(KeyAuthToken, creds.Token); err != nil {
			return err
		}
	}

	if creds.JWT != "" {
		if err := k.Set(KeyJWT, creds.JWT); err != nil {
			return err
		}
	}

	if creds.RefreshToken != "" {
		if err := k.Set(KeyRefreshToken, creds.RefreshToken); err != nil {
			return err
		}
	}

	if creds.Phone != "" {
		if err := k.Set(KeyPhone, creds.Phone); err != nil {
			return err
		}
	}

	return nil
}

// LoadCredentials loads all saved authentication credentials
func (k *Keyring) LoadCredentials() (*Credentials, error) {
	creds := &Credentials{}

	serverAddr, err := k.Get(KeyServerAddr)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	creds.ServerAddress = serverAddr

	authMethod, err := k.Get(KeyAuthMethod)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	creds.AuthMethod = authMethod

	token, err := k.Get(KeyAuthToken)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	creds.Token = token

	jwt, err := k.Get(KeyJWT)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	creds.JWT = jwt

	refreshToken, err := k.Get(KeyRefreshToken)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	creds.RefreshToken = refreshToken

	phone, err := k.Get(KeyPhone)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	creds.Phone = phone

	return creds, nil
}

// HasCredentials checks if any credentials are saved
func (k *Keyring) HasCredentials() bool {
	return k.Exists(KeyAuthToken) || k.Exists(KeyJWT)
}

// Credentials holds authentication credentials
type Credentials struct {
	ServerAddress string `json:"server_address"`
	AuthMethod    string `json:"auth_method"` // "token" or "password"
	Token         string `json:"token,omitempty"`
	JWT           string `json:"jwt,omitempty"`
	RefreshToken  string `json:"refresh_token,omitempty"`
	Phone         string `json:"phone,omitempty"`
}

// wrapError converts library errors to our error types
func wrapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
