package core

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/mephistofox/fxtun.dev/internal/inspect"
)

// maxCaptureRead is the absolute maximum bytes read into memory for a single
// request or response body during inspector capture. This prevents OOM when
// large uploads/downloads flow through an inspected tunnel. Bodies exceeding
// this limit are truncated in both the capture and the forwarded request.
const maxCaptureRead = 10 * 1024 * 1024 // 10 MB

// Capture records HTTP request/response bytes flowing through a tunnel connection.
type Capture struct {
	tunnelID     string
	tunnelName   string
	maxBodySize  int
	startTime    time.Time
	reqBuf       bytes.Buffer  // used by TeeReader path (non-HTTP fallback)
	respBuf      bytes.Buffer  // used by TeeReader path (non-HTTP fallback)
	parsedReq    *http.Request // set when request is parsed at HTTP level
	parsedResp   *http.Response
	reqBody      []byte
	reqBodySize  int64
	respBody     []byte
	respBodySize int64
}

// NewCapture creates a new capture for a single HTTP exchange.
func NewCapture(tunnelID, tunnelName string, maxBodySize int) *Capture {
	return &Capture{
		tunnelID:    tunnelID,
		tunnelName:  tunnelName,
		maxBodySize: maxBodySize,
		startTime:   time.Now(),
	}
}

// WrapRequest wraps a reader to capture request bytes. Data passes through unchanged.
// Only the first maxCaptureRead bytes are buffered for inspection; the rest
// still flows through the returned reader to the consumer.
func (c *Capture) WrapRequest(r io.Reader) io.Reader {
	return io.TeeReader(r, &limitedWriter{w: &c.reqBuf, remaining: maxCaptureRead})
}

// WrapResponse wraps a reader to capture response bytes.
// Only the first maxCaptureRead bytes are buffered for inspection.
func (c *Capture) WrapResponse(r io.Reader) io.Reader {
	return io.TeeReader(r, &limitedWriter{w: &c.respBuf, remaining: maxCaptureRead})
}

// CaptureRequest captures HTTP request metadata and body.
// Replaces req.Body so the caller can still use req.Write().
// Reads at most maxCaptureRead bytes to prevent OOM on large uploads.
func (c *Capture) CaptureRequest(req *http.Request) {
	c.parsedReq = req
	if req.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(req.Body, maxCaptureRead))
		req.Body.Close()
		c.reqBody = c.truncateBody(body)
		c.reqBodySize = int64(len(body))
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}
}

// CaptureResponse captures HTTP response metadata and body.
// Must be called BEFORE resp.Write() since Write drains the body.
// Replaces resp.Body with a new reader so the caller can still use resp.Write().
// Reads at most maxCaptureRead bytes to prevent OOM on large downloads.
func (c *Capture) CaptureResponse(resp *http.Response) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxCaptureRead))
	resp.Body.Close()
	// Store captured data.
	c.parsedResp = resp
	c.respBody = c.truncateBody(body)
	c.respBodySize = int64(len(body))
	// Replace body so resp.Write() can still send it.
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
}

// Finalize parses captured bytes into a CapturedExchange.
// Safe to call even if data is not valid HTTP — returns exchange with method UNKNOWN.
func (c *Capture) Finalize() (*inspect.CapturedExchange, error) {
	ex := &inspect.CapturedExchange{
		ID:        generateCaptureID(),
		TunnelID:  c.tunnelID,
		Timestamp: c.startTime,
		Duration:  time.Since(c.startTime),
		Method:    "UNKNOWN",
	}
	if c.parsedReq != nil {
		c.fillFromRequest(ex, c.parsedReq)
	} else {
		c.parseRequest(ex)
	}
	if c.parsedResp != nil {
		c.fillFromResponse(ex, c.parsedResp)
	} else {
		c.parseResponse(ex)
	}
	return ex, nil
}

func (c *Capture) parseRequest(ex *inspect.CapturedExchange) {
	if c.reqBuf.Len() == 0 {
		return
	}
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(c.reqBuf.Bytes())))
	if err != nil {
		// Not valid HTTP — store raw bytes as body
		ex.RequestBody = c.truncateBody(c.reqBuf.Bytes())
		ex.RequestBodySize = int64(c.reqBuf.Len())
		return
	}
	defer req.Body.Close()

	ex.Method = req.Method
	ex.Path = req.URL.RequestURI()
	ex.Host = req.Host
	ex.RequestHeaders = req.Header

	body, _ := io.ReadAll(io.LimitReader(req.Body, maxCaptureRead))
	ex.RequestBodySize = int64(len(body))
	ex.RequestBody = c.truncateBody(body)
}

func (c *Capture) parseResponse(ex *inspect.CapturedExchange) {
	if c.respBuf.Len() == 0 {
		return
	}
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(c.respBuf.Bytes())), nil)
	if err != nil {
		ex.ResponseBody = c.truncateBody(c.respBuf.Bytes())
		ex.ResponseBodySize = int64(c.respBuf.Len())
		return
	}
	defer resp.Body.Close()

	ex.StatusCode = resp.StatusCode
	ex.ResponseHeaders = resp.Header

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxCaptureRead))
	ex.ResponseBodySize = int64(len(body))
	ex.ResponseBody = c.truncateBody(body)
}

func (c *Capture) fillFromRequest(ex *inspect.CapturedExchange, req *http.Request) {
	ex.Method = req.Method
	ex.Path = req.URL.RequestURI()
	ex.Host = req.Host
	ex.RequestHeaders = req.Header
	ex.RequestBody = c.reqBody
	ex.RequestBodySize = c.reqBodySize
}

func (c *Capture) fillFromResponse(ex *inspect.CapturedExchange, resp *http.Response) {
	ex.StatusCode = resp.StatusCode
	ex.ResponseHeaders = resp.Header
	ex.ResponseBody = c.respBody
	ex.ResponseBodySize = c.respBodySize
}

func (c *Capture) truncateBody(data []byte) []byte {
	if len(data) > c.maxBodySize {
		return data[:c.maxBodySize]
	}
	return data
}

func generateCaptureID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "c-" + hex.EncodeToString(b)
}

// limitedWriter wraps an io.Writer and silently discards writes after the byte
// budget is exhausted. Used by WrapRequest/WrapResponse to cap TeeReader
// buffer growth while letting the underlying data stream continue unimpeded.
type limitedWriter struct {
	w         io.Writer
	remaining int64
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.remaining <= 0 {
		// Budget exhausted — pretend the write succeeded so TeeReader
		// keeps copying data to the real consumer.
		return len(p), nil
	}
	toWrite := p
	if int64(len(toWrite)) > lw.remaining {
		toWrite = toWrite[:lw.remaining]
	}
	n, err := lw.w.Write(toWrite)
	lw.remaining -= int64(n)
	if err != nil {
		return n, err
	}
	// Report full len(p) consumed even if we only buffered a prefix.
	// This satisfies io.TeeReader which needs Write to accept all bytes.
	return len(p), nil
}
