package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mephistofox/fxtunnel/internal/server/api/dto"
)

// newRequestDTO returns a fresh instance of the request DTO with index i.
// Every type listed here is filled straight from an unauthenticated or
// low-privilege HTTP body.
func newRequestDTO(i int) any {
	ctors := []func() any{
		func() any { return &dto.RegisterRequest{} },
		func() any { return &dto.LoginRequest{} },
		func() any { return &dto.RefreshRequest{} },
		func() any { return &dto.MagicLinkSendRequest{} },
		func() any { return &dto.MagicLinkVerifyRequest{} },
		func() any { return &dto.ChangePasswordRequest{} },
		func() any { return &dto.UpdateProfileRequest{} },
		func() any { return &dto.CreateTokenRequest{} },
		func() any { return &dto.ReserveDomainRequest{} },
		func() any { return &dto.TOTPVerifyRequest{} },
		func() any { return &dto.TOTPDisableRequest{} },
		func() any { return &dto.DeviceAuthorizeRequest{} },
		func() any { return &dto.UpdateUserRequest{} },
		func() any { return &dto.CreatePlanRequest{} },
		func() any { return &dto.UpdatePlanRequest{} },
		func() any { return &dto.MergeUsersRequest{} },
		func() any { return &dto.ResetPasswordRequest{} },
		func() any { return &dto.CheckoutRequest{} },
		func() any { return &dto.ChangePlanRequest{} },
		func() any { return &dto.ExtendSubscriptionRequest{} },
		func() any { return &dto.GrantSubscriptionRequest{} },
		func() any { return &dto.ReplayExchangeRequest{} },
		func() any { return &dto.BulkUsersRequest{} },
		func() any { return &dto.BulkTunnelsCloseRequest{} },
		func() any { return &dto.CreateInviteCodeRequest{} },
	}
	return ctors[i%len(ctors)]()
}

// FuzzDecodeAndValidate feeds arbitrary bodies through the shared decode +
// struct-validation entry point every write endpoint uses.
func FuzzDecodeAndValidate(f *testing.F) {
	f.Add(0, []byte(`{"phone":"+70000000000","password":"12345678"}`))
	f.Add(3, []byte(`{"email":"a@b.co","lang":"ru"}`))
	f.Add(8, []byte(`{"subdomain":"абвгд"}`))
	f.Add(7, []byte(`{"name":"x","max_tunnels":-2147483648}`))
	f.Add(13, []byte(`{"price":1e308,"max_tunnels":9223372036854775807}`))
	f.Add(21, []byte(`{"headers":{"a":["\u0000"]},"body":"!!!!"}`))
	f.Add(0, []byte(`{`))
	f.Add(0, []byte(`[[[[[[[[[[[[[[[[[[[[]]]]]]]]]]]]]]]]]]]]`))
	f.Add(0, []byte{})

	f.Fuzz(func(t *testing.T, idx int, body []byte) {
		if idx < 0 {
			idx = -idx
		}
		if idx < 0 { // math.MinInt
			t.Skip()
		}
		dst := newRequestDTO(idx)
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		w := httptest.NewRecorder()
		if !decodeAndValidate(w, r, dst) {
			if w.Code != http.StatusBadRequest {
				t.Fatalf("rejected body answered with %d, want 400", w.Code)
			}
			return
		}
		if w.Code != http.StatusOK {
			t.Fatalf("accepted body already wrote status %d", w.Code)
		}
	})
}

// FuzzValidateReserveDomain pins the subdomain-reservation rule: whatever the
// validator accepts is written into the reserved-domain table and served as a
// DNS label, so it must be a plain alphanumeric label of 3..32 bytes.
func FuzzValidateReserveDomain(f *testing.F) {
	f.Add("myapp")
	f.Add("ab")
	f.Add("ABC")
	f.Add("a-b")
	f.Add("абв")
	f.Add("a\u0000b")
	f.Add("１２３")

	f.Fuzz(func(t *testing.T, subdomain string) {
		req := dto.ReserveDomainRequest{Subdomain: subdomain}
		if err := validate.Struct(&req); err != nil {
			return
		}
		if len(subdomain) < 3 || len(subdomain) > 32 {
			t.Fatalf("accepted %q of length %d", subdomain, len(subdomain))
		}
		for i := 0; i < len(subdomain); i++ {
			c := subdomain[i]
			ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
			if !ok {
				t.Fatalf("accepted %q with non-alphanumeric byte %q", subdomain, c)
			}
		}
	})
}
