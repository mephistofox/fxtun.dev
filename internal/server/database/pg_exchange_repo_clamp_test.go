package database

import (
	"math"
	"testing"
)

// The stored size is not what was captured: when a backend reports a larger
// Content-Length than the body we kept, that header value is stored. A backend
// answering Content-Length: 3000000000 used to wrap to a negative int32 and put
// nonsense in the dashboard.
func TestClampBodySize(t *testing.T) {
	cases := map[int64]int32{
		0:             0,
		1024:          1024,
		-1:            0,
		3000000000:    math.MaxInt32,
		math.MaxInt64: math.MaxInt32,
		math.MaxInt32: math.MaxInt32,
	}
	for in, want := range cases {
		if got := clampBodySize(in); got != want {
			t.Errorf("clampBodySize(%d) = %d, want %d", in, got, want)
		}
	}
}
