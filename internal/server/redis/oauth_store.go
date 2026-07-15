package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mephistofox/fxtun.dev/internal/server/store"
)

var _ store.OAuthStore = (*OAuthStore)(nil)

const (
	oauthStateTTL = 10 * time.Minute
	oauthCodeTTL  = 2 * time.Minute
)

// consumeScript atomically reads and deletes a hash key.
var consumeScript = redis.NewScript(`
local data = redis.call('HGETALL', KEYS[1])
if #data > 0 then redis.call('DEL', KEYS[1]) end
return data
`)

// OAuthStore implements store.OAuthStore backed by Redis.
type OAuthStore struct {
	c *Client
}

// NewOAuthStore creates a new Redis-backed OAuth store.
func NewOAuthStore(c *Client) *OAuthStore {
	return &OAuthStore{c: c}
}

// CreateState stores OAuth state and returns a nonce.
func (o *OAuthStore) CreateState(entry *store.OAuthStateEntry) (string, error) {
	ctx := context.Background()

	nonce, err := randomHex(16)
	if err != nil {
		return "", err
	}

	key := o.c.Key("oauth", "state", nonce)
	fields := map[string]interface{}{
		"purpose":          entry.Purpose,
		"user_id":          strconv.FormatInt(entry.UserID, 10),
		"desktop_redirect": entry.DesktopRedirect,
	}

	pipe := o.c.RDB().Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, oauthStateTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return nonce, nil
}

// ConsumeState atomically retrieves and deletes an OAuth state entry.
// Returns nil if the nonce is not found or expired.
func (o *OAuthStore) ConsumeState(nonce string) *store.OAuthStateEntry {
	ctx := context.Background()
	key := o.c.Key("oauth", "state", nonce)

	result, err := consumeScript.Run(ctx, o.c.RDB(), []string{key}).StringSlice()
	if err != nil || len(result) == 0 {
		return nil
	}

	vals := sliceToMap(result)
	if len(vals) == 0 {
		return nil
	}

	userID, _ := strconv.ParseInt(vals["user_id"], 10, 64)

	return &store.OAuthStateEntry{
		Purpose:         vals["purpose"],
		UserID:          userID,
		DesktopRedirect: vals["desktop_redirect"],
	}
}

// CreateCode stores an authorization code bundle and returns the code.
func (o *OAuthStore) CreateCode(accessToken, refreshToken string, expiresIn int64) (string, error) {
	ctx := context.Background()

	code, err := randomHex(32)
	if err != nil {
		return "", err
	}

	key := o.c.Key("oauth", "code", code)
	fields := map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    strconv.FormatInt(expiresIn, 10),
	}

	pipe := o.c.RDB().Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, oauthCodeTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return code, nil
}

// ExchangeCode atomically retrieves and deletes an authorization code.
// Returns nil if the code is not found or expired.
func (o *OAuthStore) ExchangeCode(code string) *store.OAuthCodeEntry {
	ctx := context.Background()
	key := o.c.Key("oauth", "code", code)

	result, err := consumeScript.Run(ctx, o.c.RDB(), []string{key}).StringSlice()
	if err != nil || len(result) == 0 {
		return nil
	}

	vals := sliceToMap(result)
	if len(vals) == 0 {
		return nil
	}

	expiresIn, _ := strconv.ParseInt(vals["expires_in"], 10, 64)

	return &store.OAuthCodeEntry{
		AccessToken:  vals["access_token"],
		RefreshToken: vals["refresh_token"],
		ExpiresIn:    expiresIn,
	}
}

// randomHex generates n random bytes and returns them as a hex string.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sliceToMap converts a flat [k1, v1, k2, v2, ...] slice to a map.
func sliceToMap(s []string) map[string]string {
	m := make(map[string]string, len(s)/2)
	for i := 0; i+1 < len(s); i += 2 {
		m[s[i]] = s[i+1]
	}
	return m
}
