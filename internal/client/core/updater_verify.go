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
//	go build -ldflags "-X 'github.com/mephistofox/fxtun.dev/internal/client/core.updatePublicKeyHex=<hex>'"
//
// While empty, signature verification is skipped so updates keep working until
// release signing is provisioned. Once a key is set, an update with a missing
// or invalid signature is rejected — defending against a compromised server
// serving a malicious binary (which the server's own key cannot forge).
var updatePublicKeyHex = ""

// updateSignatureConfigured reports whether an update public key is baked in.
func updateSignatureConfigured() bool {
	return strings.TrimSpace(updatePublicKeyHex) != ""
}

// verifyBinarySignature checks that sigHex is a valid ed25519 signature over
// binary for pubKeyHex. An empty pubKeyHex disables verification (returns nil).
func verifyBinarySignature(binary []byte, sigHex, pubKeyHex string) error {
	pubKeyHex = strings.TrimSpace(pubKeyHex)
	if pubKeyHex == "" {
		return nil // verification disabled until a signing key is provisioned
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
