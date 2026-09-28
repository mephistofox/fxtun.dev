// cmd/client/credentials.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/mephistofox/fxtun.dev/internal/client/keyring"
	"github.com/mephistofox/fxtun.dev/internal/config"
)

// envToken is the environment variable holding the API token on headless
// servers. The config loader also accepts it (next to FXTUNNEL_SERVER_TOKEN);
// the management commands read it via os.Getenv.
const envToken = "FXTUNNEL_TOKEN"

// envServerAddr holds the server address. It matches viper's derived key for
// server.address (prefix FXTUNNEL + "server.address" with "." -> "_"), so it
// stays consistent with -c config loading. A bare FXTUNNEL_SERVER must NOT be
// used: viper would bind it to the top-level "server" map and clobber it.
const envServerAddr = "FXTUNNEL_SERVER_ADDRESS"

// checkAuth resolves credentials for management/API commands.
// Resolution order (headless-friendly first):
//  1. --token / --server flags
//  2. FXTUNNEL_TOKEN / FXTUNNEL_SERVER_ADDRESS environment variables
//  3. system keyring
//  4. ~/.fxtunnel/client.yaml
//
// Returns (token, serverAddress, ok).
func checkAuth() (string, string, bool) {
	// 1. Explicit --token flag - works without a keyring.
	if token != "" {
		return token, resolveServerAddr(), true
	}

	// 2. FXTUNNEL_TOKEN environment variable (CI / headless servers).
	if envTok := os.Getenv(envToken); envTok != "" {
		return envTok, resolveServerAddr(), true
	}

	// 3-4. Saved login, but only for the server it was saved for: an explicit
	// --server / FXTUNNEL_SERVER_ADDRESS naming another server gets no token.
	if tok, addr := savedLogin(); tok != "" {
		if a := resolveServerAddr(); a != "" {
			if !loginMatches(addr, a) {
				return "", "", false
			}
			addr = a
		}
		return tok, addr, true
	}

	return "", "", false
}

// loginMatches reports whether addr is the server a login saved for savedAddr
// belongs to. An empty saved address means the public server. A saved token
// must never reach any other server: a project file or flag naming a
// different host would otherwise receive the user's SaaS token.
func loginMatches(savedAddr, addr string) bool {
	saved := normalizeServerAddr(savedAddr)
	if config.IsPublicAddress(saved) {
		return config.IsPublicAddress(normalizeServerAddr(addr))
	}
	return saved == normalizeServerAddr(addr)
}

// savedLogin returns the token and address saved by `fxtunnel login`: the
// keyring, else the keyring-less ~/.fxtunnel/client.yaml. They come as a pair
// so a token is never matched with an address from a different login.
func savedLogin() (string, string) {
	if creds, err := keyring.New().LoadCredentials(); err == nil && creds.Token != "" {
		return creds.Token, creds.ServerAddress
	}
	tok, addr, _ := credentialsFromFile()
	return tok, addr
}

// resolveServerAddr picks the server address for flag/env token auth:
// the --server flag, else FXTUNNEL_SERVER_ADDRESS. An empty result means the
// caller should fall back to its default endpoint.
func resolveServerAddr() string {
	if serverAddr != "" {
		return serverAddr
	}
	return os.Getenv(envServerAddr)
}

// clientConfigPath returns the path to ~/.fxtunnel/client.yaml, or "" if the
// home directory cannot be determined.
func clientConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".fxtunnel", "client.yaml")
}

// credentialsFromFile reads the token and address from ~/.fxtunnel/client.yaml.
// The address is "" unless the file names one: it is read raw, without the
// config loader's defaults and env. Returns ok=false when the file is absent,
// unreadable, or holds no token.
func credentialsFromFile() (string, string, bool) {
	path := clientConfigPath()
	if path == "" {
		return "", "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	var f struct {
		Server struct{ Token, Address string }
	}
	if err := yaml.Unmarshal(data, &f); err != nil || f.Server.Token == "" {
		return "", "", false
	}
	return f.Server.Token, f.Server.Address, true
}

// saveCredentialsToFile persists the token (and optional server address) to
// ~/.fxtunnel/client.yaml. It is the fallback used when the system keyring is
// unavailable, e.g. on headless Linux without a Secret Service implementation.
// Existing settings (tunnels, inspect, etc.) in the file are preserved.
// Returns the path written.
func saveCredentialsToFile(t, serverAddress string) (string, error) {
	path := clientConfigPath()
	if path == "" {
		return "", fmt.Errorf("cannot determine home directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}

	// Preserve any existing settings by merging into the current document.
	root := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &root); err != nil {
			return "", fmt.Errorf("existing %s is corrupt: %w", path, err)
		}
	}
	server, _ := root["server"].(map[string]any)
	if server == nil {
		server = map[string]any{}
	}
	server["token"] = t
	if serverAddress != "" {
		server["address"] = serverAddress
	}
	root["server"] = server

	data, err := yaml.Marshal(root)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", err
	}
	return path, nil
}

// clearCredentialsFile removes the saved token and server address from
// ~/.fxtunnel/client.yaml while preserving other settings. It is a no-op when
// the file (or its server section) does not exist.
func clearCredentialsFile() error {
	path := clientConfigPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	root := map[string]any{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("existing %s is corrupt: %w", path, err)
	}
	server, ok := root["server"].(map[string]any)
	if !ok {
		return nil
	}
	delete(server, "token")
	delete(server, "address")
	if len(server) == 0 {
		delete(root, "server")
	} else {
		root["server"] = server
	}
	out, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0600)
}

// persistCredentials saves the token to the system keyring, falling back to a
// plaintext config file when the keyring is unavailable. It prints a
// user-facing confirmation describing where the token was stored.
func persistCredentials(t, serverAddress string) error {
	kr := keyring.New()
	creds := keyring.Credentials{
		Token:      t,
		AuthMethod: "token",
	}
	if serverAddress != "" {
		creds.ServerAddress = serverAddress
	}
	if err := kr.SaveCredentials(creds); err != nil {
		// Keyring unavailable (headless Linux without Secret Service, etc.).
		path, ferr := saveCredentialsToFile(t, serverAddress)
		if ferr != nil {
			return fmt.Errorf("keyring unavailable (%v) and file fallback failed: %w", err, ferr)
		}
		fmt.Printf("System keyring unavailable - token saved to %s instead.\n", path)
		return nil
	}
	fmt.Println("Token saved. You can now use fxtunnel without --token flag.")
	return nil
}
