package tls

import (
	"strings"
	"testing"
)

// reserved mirrors what the API passes in: the base domain, its aliases and
// every authoritative DNS zone this installation owns.
var reserved = []string{"fxtun.dev", "fxtun.ru", "fxcode.ru"}

// FuzzValidateCustomDomain drives the custom-domain gate with arbitrary input.
//
// Whatever it accepts is stored, served from the TLS SNI routing map and later
// handed to ACME, so an accepted value must be a syntactically valid hostname
// and must never resolve into a name this installation already owns.
func FuzzValidateCustomDomain(f *testing.F) {
	f.Add("example.com")
	f.Add("sub.example.com")
	f.Add("FXTUN.DEV")
	f.Add("x.fxtun.dev")
	f.Add("1.2.3.4")
	f.Add("localhost")
	f.Add("a.localhost")
	f.Add("..")
	f.Add("*.example.com")
	f.Add("a b.com")
	f.Add("a\n.com")
	f.Add("-.-")
	f.Add("пример.рф")
	f.Add(strings.Repeat("a.", 200) + "com")

	f.Fuzz(func(t *testing.T, domain string) {
		norm := NormalizeDomain(domain)
		if NormalizeDomain(norm) != norm {
			t.Fatalf("NormalizeDomain is not idempotent: %q -> %q -> %q", domain, norm, NormalizeDomain(norm))
		}
		if err := ValidateCustomDomain(norm, reserved...); err != nil {
			return
		}

		if len(norm) == 0 || len(norm) > 253 {
			t.Fatalf("accepted %q: length %d is not a valid hostname length", norm, len(norm))
		}
		for _, label := range strings.Split(norm, ".") {
			if label == "" || len(label) > 63 {
				t.Fatalf("accepted %q: bad label %q", norm, label)
			}
			if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
				t.Fatalf("accepted %q: label %q starts or ends with a hyphen", norm, label)
			}
			for i := 0; i < len(label); i++ {
				c := label[i]
				ok := c == '-' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
				if !ok {
					t.Fatalf("accepted %q: label %q has illegal byte %q", norm, label, c)
				}
			}
		}
		for _, r := range reserved {
			if norm == r || strings.HasSuffix(norm, "."+r) {
				t.Fatalf("accepted %q, which is inside reserved domain %q", norm, r)
			}
		}
		// The challenge record name is fed to a DNS resolver; it must stay a
		// name, not a smuggled second query.
		if strings.ContainsAny(ChallengeRecordName(norm), " \t\r\n\x00") {
			t.Fatalf("challenge name for %q is not a bare DNS name", norm)
		}
	})
}

// FuzzIsApexDomain checks the apex classifier that decides whether ownership
// is proven by A/AAAA or by CNAME.
func FuzzIsApexDomain(f *testing.F) {
	f.Add("example.com")
	f.Add("a.example.com")
	f.Add(".")
	f.Add("")

	f.Fuzz(func(t *testing.T, domain string) {
		if IsApexDomain(domain) && strings.Count(domain, ".") != 1 {
			t.Fatalf("%q classified as apex but has %d dots", domain, strings.Count(domain, "."))
		}
	})
}
