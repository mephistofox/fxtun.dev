package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoInsecureTLS pins that no GUI code path disables TLS certificate
// verification. Every GUI request carries the user's JWT in an Authorization
// header, so a skipped verification hands that token to any MITM.
func TestNoInsecureTLS(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "InsecureSkipVerify") {
			t.Errorf("%s disables TLS verification", f)
		}
	}
}
