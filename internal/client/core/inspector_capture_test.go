package core

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureHTTPExchange(t *testing.T) {
	rawReq := "POST /api/webhook HTTP/1.1\r\n" +
		"Host: myapp.fxtun.dev\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n" +
		"{\"key\":\"val\"}"

	rawResp := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 15\r\n" +
		"\r\n" +
		"{\"status\":\"ok\"}"

	cap := NewCapture("tun-1", "myapp", 4096)

	// Wrap readers — data must pass through unchanged.
	reqReader := cap.WrapRequest(strings.NewReader(rawReq))
	respReader := cap.WrapResponse(strings.NewReader(rawResp))

	// Read all data through the wrapped readers.
	reqData, err := io.ReadAll(reqReader)
	require.NoError(t, err)
	assert.Equal(t, rawReq, string(reqData), "request data must pass through unchanged")

	respData, err := io.ReadAll(respReader)
	require.NoError(t, err)
	assert.Equal(t, rawResp, string(respData), "response data must pass through unchanged")

	// Finalize and verify parsed exchange.
	ex, err := cap.Finalize()
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(ex.ID, "c-"), "ID should start with c-")
	assert.Equal(t, "tun-1", ex.TunnelID)
	assert.False(t, ex.Timestamp.IsZero())
	assert.True(t, ex.Duration >= 0)

	// Request fields.
	assert.Equal(t, "POST", ex.Method)
	assert.Equal(t, "/api/webhook", ex.Path)
	assert.Equal(t, "myapp.fxtun.dev", ex.Host)
	assert.Equal(t, "application/json", ex.RequestHeaders.Get("Content-Type"))
	assert.Equal(t, []byte("{\"key\":\"val\"}"), ex.RequestBody)
	assert.Equal(t, int64(13), ex.RequestBodySize)

	// Response fields.
	assert.Equal(t, 200, ex.StatusCode)
	assert.Equal(t, "application/json", ex.ResponseHeaders.Get("Content-Type"))
	assert.Equal(t, []byte("{\"status\":\"ok\"}"), ex.ResponseBody)
	assert.Equal(t, int64(15), ex.ResponseBodySize)
}

func TestCaptureBodyTruncation(t *testing.T) {
	maxBody := 1024
	bigBody := bytes.Repeat([]byte("X"), 2048)

	rawReq := "POST /upload HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Length: 2048\r\n" +
		"\r\n" +
		string(bigBody)

	cap := NewCapture("tun-2", "example", maxBody)

	reqReader := cap.WrapRequest(strings.NewReader(rawReq))
	_, err := io.ReadAll(reqReader)
	require.NoError(t, err)

	ex, err := cap.Finalize()
	require.NoError(t, err)

	// Body should be truncated to maxBodySize.
	assert.Len(t, ex.RequestBody, maxBody, "body should be truncated to maxBodySize")

	// RequestBodySize should reflect the actual full body size.
	assert.Equal(t, int64(2048), ex.RequestBodySize, "RequestBodySize should reflect actual size")
}

func TestCaptureBinaryNonHTTP(t *testing.T) {
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x89, 0x50, 0x4E, 0x47}

	cap := NewCapture("tun-3", "binary", 4096)

	reqReader := cap.WrapRequest(bytes.NewReader(binaryData))
	passedThrough, err := io.ReadAll(reqReader)
	require.NoError(t, err)
	assert.Equal(t, binaryData, passedThrough, "binary data must pass through unchanged")

	ex, err := cap.Finalize()
	require.NoError(t, err)

	// Should not panic; method stays UNKNOWN for non-HTTP data.
	assert.Equal(t, "UNKNOWN", ex.Method)

	// Raw bytes stored as request body.
	assert.Equal(t, binaryData, ex.RequestBody)
	assert.Equal(t, int64(len(binaryData)), ex.RequestBodySize)

	// No response was captured.
	assert.Equal(t, 0, ex.StatusCode)
	assert.Nil(t, ex.ResponseHeaders)
	assert.Nil(t, ex.ResponseBody)
}

// The body must pass through untouched and unbuffered: size, Content-Length
// and every byte preserved, only the first maxBodySize bytes kept for display.
func TestCaptureResponseStreamsBody(t *testing.T) {
	const size = 20 << 20
	body := bytes.Repeat([]byte("x"), size)
	resp := &http.Response{
		StatusCode:    200,
		Header:        http.Header{},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: size,
	}
	c := NewCapture("t", "n", 256*1024)
	c.CaptureResponse(resp)
	if resp.ContentLength != size {
		t.Fatalf("ContentLength rewritten to %d", resp.ContentLength)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil || len(got) != size {
		t.Fatalf("forwarded %d bytes, err=%v", len(got), err)
	}
	resp.Body.Close()
	ex, _ := c.Finalize()
	if len(ex.ResponseBody) != 256*1024 || ex.ResponseBodySize != size {
		t.Fatalf("captured %d bytes, size %d", len(ex.ResponseBody), ex.ResponseBodySize)
	}
}

// Capture must not read ahead: a reader that blocks after the first chunk
// still yields that chunk to the consumer immediately.
func TestCaptureResponseDoesNotReadAhead(t *testing.T) {
	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: pr, ContentLength: -1}
	c := NewCapture("t", "n", 1024)
	done := make(chan struct{})
	go func() {
		c.CaptureResponse(resp)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("CaptureResponse blocked reading the body")
	}
	go pw.Write([]byte("data: 1\n\n"))
	buf := make([]byte, 64)
	n, err := resp.Body.Read(buf)
	if err != nil || string(buf[:n]) != "data: 1\n\n" {
		t.Fatalf("read %q, err=%v", buf[:n], err)
	}
	pw.Close()
}

func TestCaptureRequestStreamsBody(t *testing.T) {
	const size = 12 << 20
	req, _ := http.NewRequest("POST", "http://x/", bytes.NewReader(bytes.Repeat([]byte("y"), size)))
	c := NewCapture("t", "n", 1024)
	c.CaptureRequest(req)
	if req.ContentLength != size {
		t.Fatalf("ContentLength rewritten to %d", req.ContentLength)
	}
	got, _ := io.ReadAll(req.Body)
	if len(got) != size {
		t.Fatalf("forwarded %d bytes, want %d (no truncation)", len(got), size)
	}
	if c.reqBodySize != size {
		t.Fatalf("reqBodySize=%d", c.reqBodySize)
	}
}
