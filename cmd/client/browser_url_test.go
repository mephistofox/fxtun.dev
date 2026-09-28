package main

import "testing"

// The device-flow URL arrives in the server's JSON response and on Windows is
// handed to `cmd /c start`, which Go leaves unquoted unless it contains spaces
// or quotes — so a shell metacharacter in it executes on the user's machine.
func TestSafeBrowserURL(t *testing.T) {
	hostile := []string{
		"https://x/?a=1&calc",
		"https://x/?a=1|calc",
		"https://x/?a=1;calc",
		"https://x/?a=$(id)",
		"https://x/?a=`id`",
		"https://x/ && calc",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"not a url",
		"",
	}
	for _, u := range hostile {
		if safeBrowserURL(u) {
			t.Errorf("safeBrowserURL(%q) = true, want false", u)
		}
	}

	for _, u := range []string{
		"https://fxtun.ru/auth/cli",
		"http://127.0.0.1:4040/",
		"https://fxtun.ru/auth/cli?session=abc123",
	} {
		if !safeBrowserURL(u) {
			t.Errorf("safeBrowserURL(%q) = false, want true", u)
		}
	}
}
