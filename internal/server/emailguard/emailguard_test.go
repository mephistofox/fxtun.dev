package emailguard

import (
	"context"
	"errors"
	"net"
	"testing"
)

// fakeResolver lets tests control MX/host lookups without real DNS.
type fakeResolver struct {
	mx    map[string][]*net.MX
	hosts map[string][]string
}

func (f *fakeResolver) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	if v, ok := f.mx[name]; ok {
		return v, nil
	}
	return nil, errors.New("no mx")
}

func (f *fakeResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if v, ok := f.hosts[host]; ok {
		return v, nil
	}
	return nil, errors.New("no host")
}

func newTestValidator(r resolver) *Validator {
	v := New()
	v.resolver = r
	return v
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		in       string
		wantNorm string
		wantErr  bool
	}{
		{"  User@Example.COM ", "User@example.com", false},
		{"a@b.co", "a@b.co", false},
		{"no-at-sign", "", true},
		{"@example.com", "", true},
		{"user@", "", true},
		{"user@nodot", "", true},
		{"user@example..com", "", true},
		{"user name@example.com", "", true},
		{"user@exa_mple.com", "", true},
	}
	for _, c := range cases {
		got, _, err := Normalize(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("Normalize(%q) expected error, got %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Normalize(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.wantNorm {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.wantNorm)
		}
	}
}

func TestIsDisposable(t *testing.T) {
	v := New()
	disposable := []string{"mailinator.com", "sub.mailinator.com", "TEMP-MAIL.org", "guerrillamail.com"}
	for _, d := range disposable {
		if !v.IsDisposable(d) {
			t.Errorf("IsDisposable(%q) = false, want true", d)
		}
	}
	legit := []string{"gmail.com", "example.com", "mycompany.io"}
	for _, d := range legit {
		if v.IsDisposable(d) {
			t.Errorf("IsDisposable(%q) = true, want false", d)
		}
	}
}

func TestValidate(t *testing.T) {
	r := &fakeResolver{
		mx:    map[string][]*net.MX{"good.com": {{Host: "mx.good.com.", Pref: 10}}},
		hosts: map[string][]string{"ahost.com": {"1.2.3.4"}},
	}
	v := newTestValidator(r)
	ctx := context.Background()

	if _, err := v.Validate(ctx, "user@good.com"); err != nil {
		t.Errorf("Validate(good MX) unexpected error: %v", err)
	}
	// No MX but has A record → deliverable per RFC 5321.
	if _, err := v.Validate(ctx, "user@ahost.com"); err != nil {
		t.Errorf("Validate(A fallback) unexpected error: %v", err)
	}
	// Disposable must be rejected before any DNS work.
	if _, err := v.Validate(ctx, "user@mailinator.com"); !errors.Is(err, ErrDisposableEmail) {
		t.Errorf("Validate(disposable) = %v, want ErrDisposableEmail", err)
	}
	// No MX and no A/AAAA → undeliverable.
	if _, err := v.Validate(ctx, "user@nowhere.com"); !errors.Is(err, ErrUndeliverable) {
		t.Errorf("Validate(no records) = %v, want ErrUndeliverable", err)
	}
	// Malformed.
	if _, err := v.Validate(ctx, "bogus"); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("Validate(malformed) = %v, want ErrInvalidEmail", err)
	}
}

func TestValidateCaches(t *testing.T) {
	calls := 0
	r := &countingResolver{calls: &calls, inner: &fakeResolver{
		mx: map[string][]*net.MX{"good.com": {{Host: "mx.good.com.", Pref: 10}}},
	}}
	v := newTestValidator(r)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := v.Validate(ctx, "user@good.com"); err != nil {
			t.Fatalf("Validate unexpected error: %v", err)
		}
	}
	if calls != 1 {
		t.Errorf("expected 1 DNS lookup due to caching, got %d", calls)
	}
}

type countingResolver struct {
	calls *int
	inner resolver
}

func (c *countingResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	*c.calls++
	return c.inner.LookupMX(ctx, name)
}

func (c *countingResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return c.inner.LookupHost(ctx, host)
}
