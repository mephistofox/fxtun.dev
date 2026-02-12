package client

import (
	"bytes"
	"io"
	"strings"
	"testing"

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
