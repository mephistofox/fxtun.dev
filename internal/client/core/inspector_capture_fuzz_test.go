package core

import (
	"bytes"
	"testing"
)

// FuzzCaptureFinalize feeds arbitrary tunnel bytes through the inspector's
// HTTP request/response parser. Both directions carry data straight off the
// wire, and the parse result is stored and later rendered in the dashboard.
func FuzzCaptureFinalize(f *testing.F) {
	f.Add([]byte("GET / HTTP/1.1\r\nHost: x\r\n\r\n"), []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nhi"), 1024)
	f.Add([]byte("POST /p HTTP/1.1\r\nHost: x\r\nTransfer-Encoding: chunked\r\n\r\nzz\r\n"), []byte("HTTP/1.1 999 \x00\r\n\r\n"), 0)
	f.Add([]byte("GET /\x00\xff HTTP/1.1\r\nHost: \r\n\r\n"), []byte("garbage"), 8)
	f.Add([]byte{}, []byte{}, 16)
	f.Add([]byte("GET / HTTP/1.1\r\nContent-Length: 99999999999999\r\n\r\n"), []byte("HTTP/1.1 200 OK\r\nContent-Length: -1\r\n\r\n"), 4)

	f.Fuzz(func(t *testing.T, req, resp []byte, maxBody int) {
		if maxBody < 0 || maxBody > 1<<20 {
			t.Skip()
		}
		c := NewCapture("t-1", "name", maxBody)
		if _, err := c.reqBuf.Write(req); err != nil {
			t.Fatal(err)
		}
		if _, err := c.respBuf.Write(resp); err != nil {
			t.Fatal(err)
		}

		ex, err := c.Finalize()
		if err != nil {
			t.Fatalf("Finalize returned an error: %v", err)
		}
		if ex == nil {
			t.Fatal("Finalize returned no exchange")
		}
		if len(ex.RequestBody) > maxBody || len(ex.ResponseBody) > maxBody {
			t.Fatalf("captured body exceeds maxBodySize %d: %d/%d",
				maxBody, len(ex.RequestBody), len(ex.ResponseBody))
		}
	})
}

// FuzzLimitedWriter checks the TeeReader budget wrapper: it must always report
// the full write as consumed (or TeeReader errors out and the tunnel stalls)
// and must never buffer more than its budget.
func FuzzLimitedWriter(f *testing.F) {
	f.Add([]byte("hello"), int64(3))
	f.Add([]byte("hello"), int64(0))
	f.Add([]byte("hello"), int64(-1))
	f.Add([]byte{}, int64(1<<40))

	f.Fuzz(func(t *testing.T, data []byte, budget int64) {
		var buf bytes.Buffer
		lw := &limitedWriter{w: &buf, remaining: budget}
		n, err := lw.Write(data)
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if n != len(data) {
			t.Fatalf("reported %d of %d bytes consumed", n, len(data))
		}
		if budget >= 0 && int64(buf.Len()) > budget {
			t.Fatalf("buffered %d bytes with a budget of %d", buf.Len(), budget)
		}
	})
}
