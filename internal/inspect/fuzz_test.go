package inspect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// FuzzRingBuffer drives the ring buffer with an arbitrary capacity and an
// arbitrary sequence of add/list/get operations, including the out-of-range
// offsets and limits a paginating caller can pass in.
func FuzzRingBuffer(f *testing.F) {
	f.Add(uint8(4), 10, 0, 5)
	f.Add(uint8(0), 1, 0, 1)
	f.Add(uint8(1), 3, -1, -1)
	f.Add(uint8(2), 3, 1<<20, 1<<20)
	f.Add(uint8(3), 5, -1<<20, 2)

	f.Fuzz(func(t *testing.T, capacity uint8, adds, offset, limit int) {
		if adds < 0 || adds > 1000 {
			t.Skip()
		}
		rb := NewRingBuffer(int(capacity))
		for i := 0; i < adds; i++ {
			rb.Add(&CapturedExchange{ID: fmt.Sprintf("e-%d", i)})
		}

		got := rb.List(offset, limit)
		if len(got) > rb.Len() {
			t.Fatalf("List(%d,%d) returned %d entries, buffer holds %d", offset, limit, len(got), rb.Len())
		}
		if limit >= 0 && len(got) > limit {
			t.Fatalf("List(%d,%d) returned %d entries, above the limit", offset, limit, len(got))
		}
		for i, e := range got {
			if e == nil {
				t.Fatalf("List(%d,%d)[%d] is nil", offset, limit, i)
			}
		}
		_ = rb.Get(fmt.Sprintf("e-%d", offset))
		rb.Clear()
		if rb.Len() != 0 {
			t.Fatal("Clear left entries behind")
		}
	})
}

// FuzzExchangeJSON round-trips a captured exchange through the JSON encoding
// used to hand it to the dashboard and to the persistence layer.
func FuzzExchangeJSON(f *testing.F) {
	f.Add("GET", "/", "example.com", []byte("hi"), "X-A", "1")
	f.Add("\x00", "/\xff\xfe", "", []byte{0xff, 0x00}, "\r\nX: y", "\r\n")
	f.Add("POST", "/x", "h", []byte{}, "", "")

	f.Fuzz(func(t *testing.T, method, path, host string, body []byte, hk, hv string) {
		ex := &CapturedExchange{
			ID:              "id",
			Method:          method,
			Path:            path,
			Host:            host,
			RequestBody:     body,
			RequestBodySize: int64(len(body)),
			RequestHeaders:  http.Header{hk: []string{hv}},
			ResponseHeaders: http.Header{},
		}
		data, err := json.Marshal(ex)
		if err != nil {
			t.Fatalf("captured exchange is not encodable: %v", err)
		}
		var back CapturedExchange
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("our own encoding does not decode: %v", err)
		}
		// The summary is what the list endpoint returns; it must encode too.
		if _, err := json.Marshal(back.Summary()); err != nil {
			t.Fatalf("summary is not encodable: %v", err)
		}
	})
}
