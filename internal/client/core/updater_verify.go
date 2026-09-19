package core

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"
)

// updatePublicKeyHex is the hex-encoded ed25519 public key used to verify
// self-update binaries. It is intentionally empty by default and baked in at
// build time via ldflags, e.g.:
//
//	go build -ldflags "-X 'github.com/mephistofox/fxtunnel/internal/client/core.updatePublicKeyHex=<hex>'"
//
// Every released build (CLI and GUI, production and staging) bakes the key in;
// only a plain local `go build` leaves it empty, and there verification is
// skipped. Once a key is set, an update with a missing or invalid signature is
// rejected — defending against a compromised server serving a malicious binary
// (which the server's own key cannot forge).
var updatePublicKeyHex = ""

// updateSignatureConfigured reports whether an update public key is baked in.
func updateSignatureConfigured() bool {
	return strings.TrimSpace(updatePublicKeyHex) != ""
}

// verifyBinarySignature checks that sigHex is a valid ed25519 signature over
// binary for pubKeyHex. It fails closed: an empty key is an error, never a
// pass. Whether verification runs at all is decided solely by
// updateSignatureConfigured, so there is one place to get that wrong.
func verifyBinarySignature(binary []byte, sigHex, pubKeyHex string) error {
	pubKeyHex = strings.TrimSpace(pubKeyHex)
	if pubKeyHex == "" {
		return fmt.Errorf("no update public key configured")
	}

	pub, err := hex.DecodeString(pubKeyHex)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid update public key")
	}

	sig, err := hex.DecodeString(strings.TrimSpace(sigHex))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("invalid update signature")
	}

	if !ed25519.Verify(ed25519.PublicKey(pub), binary, sig) {
		return fmt.Errorf("update signature verification failed")
	}
	return nil
}
