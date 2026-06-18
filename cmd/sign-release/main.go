// Command sign-release signs release binaries with the fxTunnel update signing
// key and writes <file>.sig next to each, containing the hex-encoded ed25519
// signature that the client verifies in internal/client/core before applying a
// self-update.
//
// The signing key is read from FXTUNNEL_UPDATE_SIGNING_KEY as a 32-byte hex
// ed25519 seed (the private half; keep it as a CI secret, never commit it).
//
//	FXTUNNEL_UPDATE_SIGNING_KEY=<seed-hex> sign-release fxtunnel-linux-amd64 ...
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sign-release <file> [<file>...]")
		os.Exit(2)
	}

	seedHex := strings.TrimSpace(os.Getenv("FXTUNNEL_UPDATE_SIGNING_KEY"))
	if seedHex == "" {
		fmt.Fprintln(os.Stderr, "FXTUNNEL_UPDATE_SIGNING_KEY is not set")
		os.Exit(1)
	}

	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			os.Exit(1)
		}
		sigHex, err := signData(seedHex, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sign %s: %v\n", path, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path+".sig", []byte(sigHex), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s.sig: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("signed %s -> %s.sig\n", path, path)
	}
}

// signData returns the hex-encoded ed25519 signature over data using a 32-byte
// hex seed.
func signData(seedHex string, data []byte) (string, error) {
	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		return "", fmt.Errorf("signing key is not valid hex")
	}
	if len(seed) != ed25519.SeedSize {
		return "", fmt.Errorf("signing key must be a %d-byte hex seed, got %d bytes", ed25519.SeedSize, len(seed))
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return hex.EncodeToString(ed25519.Sign(priv, data)), nil
}
