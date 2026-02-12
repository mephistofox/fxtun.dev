package client

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/mephistofox/fxtunnel/internal/inspect"
)

// Capture records HTTP request/response bytes flowing through a tunnel connection.
type Capture struct {
	tunnelID    string
	tunnelName  string
	maxBodySize int
	startTime   time.Time
	reqBuf      bytes.Buffer
	respBuf     bytes.Buffer
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
	c.parseRequest(ex)
	c.parseResponse(ex)
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
