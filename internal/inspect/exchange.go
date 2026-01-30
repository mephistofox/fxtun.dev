package inspect

import (
	"net/http"
	"time"
)

const MaxBodySize = 256 * 1024

type CapturedExchange struct {
	ID        string        `json:"id"`
	TunnelID  string        `json:"tunnel_id"`
	TraceID   string        `json:"trace_id,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration_ns"`

	Method          string      `json:"method"`
	Path            string      `json:"path"`
	Host            string      `json:"host"`
	RequestHeaders  http.Header `json:"request_headers"`
	RequestBody     []byte      `json:"request_body,omitempty"`
	RequestBodySize int64       `json:"request_body_size"`
	RemoteAddr      string      `json:"remote_addr"`

	StatusCode       int         `json:"status_code"`
	ResponseHeaders  http.Header `json:"response_headers"`
	ResponseBody     []byte      `json:"response_body,omitempty"`
	ResponseBodySize int64       `json:"response_body_size"`
}

type ExchangeSummary struct {
	ID               string        `json:"id"`
	TunnelID         string        `json:"tunnel_id"`
	TraceID          string        `json:"trace_id,omitempty"`
	Timestamp        time.Time     `json:"timestamp"`
	Duration         time.Duration `json:"duration_ns"`
	Method           string        `json:"method"`
	Path             string        `json:"path"`
	Host             string        `json:"host"`
	StatusCode       int           `json:"status_code"`
	RequestBodySize  int64         `json:"request_body_size"`
	ResponseBodySize int64         `json:"response_body_size"`
	RemoteAddr       string        `json:"remote_addr"`
}

func (e *CapturedExchange) Summary() ExchangeSummary {
	return ExchangeSummary{
		ID: e.ID, TunnelID: e.TunnelID, TraceID: e.TraceID, Timestamp: e.Timestamp, Duration: e.Duration,
		Method: e.Method, Path: e.Path, Host: e.Host, StatusCode: e.StatusCode,
		RequestBodySize: e.RequestBodySize, ResponseBodySize: e.ResponseBodySize,
		RemoteAddr: e.RemoteAddr,
	}
}
