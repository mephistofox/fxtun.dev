package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func TestSignData(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	seedHex := hex.EncodeToString(priv.Seed())
	data := []byte("pretend release binary")

	sigHex, err := signData(seedHex, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != ed25519.SignatureSize {
		t.Fatalf("signature is not %d-byte hex: %v", ed25519.SignatureSize, err)
	}

	// The signature must verify against the public key with the same primitive
	// the client uses (ed25519.Verify over the raw bytes).
	if !ed25519.Verify(pub, data, sig) {
		t.Fatal("signature must verify with the derived public key")
	}
	if ed25519.Verify(pub, []byte("tampered"), sig) {
		t.Fatal("signature must not verify over tampered data")
	}
}

func TestSignData_BadSeed(t *testing.T) {
	if _, err := signData("not-hex", []byte("x")); err == nil {
		t.Fatal("expected error on non-hex seed")
	}
	if _, err := signData(hex.EncodeToString([]byte("too-short")), []byte("x")); err == nil {
		t.Fatal("expected error on wrong-length seed")
	}
}
