package emailguard

import (
	"strings"
	"testing"
)

// FuzzNormalize drives the address normalizer with arbitrary input.
//
// The normalized address is the identity everything downstream keys on —
// user rows, rate-limit buckets, the per-recipient send cooldown — so the
// invariants that matter are: it is idempotent (normalizing twice changes
// nothing, otherwise two spellings of one address get two buckets), it stays
// within the 254-byte address limit, and it never carries control characters
// into an outgoing mail header.
func FuzzNormalize(f *testing.F) {
	f.Add("user@example.com")
	f.Add("  User@Example.COM  ")
	f.Add("a@b")
	f.Add("@example.com")
	f.Add("user@")
	f.Add("u@ser@example.com")
	f.Add("user@exa..mple.com")
	f.Add("user@.example.com")
	f.Add("İ@example.com")
	f.Add("user\x00@example.com")
	f.Add("user\rnBcc: x@y.z@example.com")
	f.Add(strings.Repeat("a", 250) + "@example.com")

	f.Fuzz(func(t *testing.T, email string) {
		normalized, domain, err := Normalize(email)
		if err != nil {
			return
		}

		if len(normalized) > 254 {
			t.Fatalf("normalized address is %d bytes (> 254): %q", len(normalized), normalized)
		}
		if !strings.HasSuffix(normalized, "@"+domain) {
			t.Fatalf("normalized %q does not end in its domain %q", normalized, domain)
		}
		for _, r := range normalized {
			if r < 0x20 || r == 0x7f {
				t.Fatalf("normalized %q carries control character %#U", normalized, r)
			}
		}

		n2, d2, err2 := Normalize(normalized)
		if err2 != nil {
			t.Fatalf("normalized %q is rejected on the second pass: %v", normalized, err2)
		}
		if n2 != normalized || d2 != domain {
			t.Fatalf("not idempotent: %q -> %q -> %q", email, normalized, n2)
		}
	})
}

// FuzzIsDisposable checks the blocklist walk against arbitrary domains.
func FuzzIsDisposable(f *testing.F) {
	f.Add("mailinator.com")
	f.Add("foo.mailinator.com")
	f.Add("")
	f.Add(".")
	f.Add(strings.Repeat(".", 4096))

	v := New()
	f.Fuzz(func(t *testing.T, domain string) {
		if v.IsDisposable(domain) {
			// A positive must be explained by the domain itself or a parent.
			d := strings.ToLower(strings.TrimSpace(domain))
			if !strings.Contains(d, ".") {
				t.Fatalf("dotless domain %q reported as disposable", domain)
			}
		}
	})
}
