package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func TestVerifyBinarySignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	pubHex := hex.EncodeToString(pub)

	binary := []byte("pretend this is a release binary")
	sigHex := hex.EncodeToString(ed25519.Sign(priv, binary))

	t.Run("valid signature passes", func(t *testing.T) {
		if err := verifyBinarySignature(binary, sigHex, pubHex); err != nil {
			t.Fatalf("expected pass, got %v", err)
		}
	})

	t.Run("tampered binary fails", func(t *testing.T) {
		if err := verifyBinarySignature([]byte("malicious binary"), sigHex, pubHex); err == nil {
			t.Fatal("expected failure on tampered binary")
		}
	})

	t.Run("wrong key fails", func(t *testing.T) {
		otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
		if err := verifyBinarySignature(binary, sigHex, hex.EncodeToString(otherPub)); err == nil {
			t.Fatal("expected failure with wrong public key")
		}
	})

	t.Run("empty public key disables verification", func(t *testing.T) {
		if err := verifyBinarySignature(binary, sigHex, ""); err != nil {
			t.Fatalf("empty key should skip verification, got %v", err)
		}
	})

	t.Run("malformed signature fails", func(t *testing.T) {
		if err := verifyBinarySignature(binary, "not-hex", pubHex); err == nil {
			t.Fatal("expected failure on malformed signature")
		}
	})

	t.Run("malformed public key fails", func(t *testing.T) {
		if err := verifyBinarySignature(binary, sigHex, "zz"); err == nil {
			t.Fatal("expected failure on malformed public key")
		}
	})

	t.Run("wrong-length key fails", func(t *testing.T) {
		if err := verifyBinarySignature(binary, sigHex, hex.EncodeToString([]byte("short"))); err == nil {
			t.Fatal("expected failure on wrong-length key")
		}
	})
}

func TestUpdateSignatureConfigured(t *testing.T) {
	if updateSignatureConfigured() {
		t.Fatal("expected verification disabled by default (empty embedded key)")
	}
}
