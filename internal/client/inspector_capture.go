package client

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
func (c *Capture) WrapRequest(r io.Reader) io.Reader {
	return io.TeeReader(r, &c.reqBuf)
}

// WrapResponse wraps a reader to capture response bytes.
func (c *Capture) WrapResponse(r io.Reader) io.Reader {
	return io.TeeReader(r, &c.respBuf)
}

// CaptureRequest captures HTTP request metadata and body.
// Replaces req.Body so the caller can still use req.Write().
func (c *Capture) CaptureRequest(req *http.Request) {
	c.parsedReq = req
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
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
func (c *Capture) CaptureResponse(resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
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

	body, _ := io.ReadAll(req.Body)
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

	body, _ := io.ReadAll(resp.Body)
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
