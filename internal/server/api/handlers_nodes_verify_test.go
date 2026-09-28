package api

import (
	"errors"
	"testing"

	"github.com/mephistofox/fxtun.dev/internal/server/database"
)

// A failed token lookup is not a rejected token: clients stop reconnecting on
// "invalid token", so a database blip on the hub must not produce one.
func TestVerifyTokenDBErrorIsNotInvalidToken(t *testing.T) {
	s := &Server{}
	failing := func(string) (*database.APIToken, error) { return nil, errors.New("connection refused") }

	resp, err := s.verifyToken("sk_whatever", failing, nil)
	if err == nil {
		t.Fatalf("got %+v and no error, want an error for the database failure", resp)
	}
}

func TestVerifyTokenUnknownTokenIsInvalid(t *testing.T) {
	s := &Server{}
	missing := func(string) (*database.APIToken, error) { return nil, database.ErrTokenNotFound }

	resp, err := s.verifyToken("sk_whatever", missing, nil)
	if err != nil || resp.Valid || resp.Error != "invalid token" {
		t.Fatalf("got %+v, %v; want invalid token", resp, err)
	}
}

// Same for the token owner's lookup: a database error is not "user not found".
func TestVerifyTokenUserDBErrorIsNotInvalid(t *testing.T) {
	s := &Server{}
	found := func(string) (*database.APIToken, error) { return &database.APIToken{UserID: 1}, nil }
	failing := func(int64) (*database.User, error) { return nil, errors.New("connection refused") }

	resp, err := s.verifyToken("sk_whatever", found, failing)
	if err == nil {
		t.Fatalf("got %+v and no error, want an error for the database failure", resp)
	}

	missing := func(int64) (*database.User, error) { return nil, database.ErrUserNotFound }
	resp, err = s.verifyToken("sk_whatever", found, missing)
	if err != nil || resp.Valid || resp.Error != "user not found" {
		t.Fatalf("got %+v, %v; want user not found", resp, err)
	}
}
