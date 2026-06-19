package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mephistofox/fxtun.dev/internal/inspect"
	"github.com/mephistofox/fxtun.dev/internal/server/database/sqlc"
)

// ExchangeRepository handles inspect exchange database operations using PostgreSQL via sqlc.
type ExchangeRepository struct {
	q *sqlc.Queries
}

const maxExchangeBodySize = 1 << 20 // 1MB

// Save persists a captured exchange to the database.
func (r *ExchangeRepository) Save(ex *inspect.CapturedExchange, userID int64) error {
	reqHeaders, err := json.Marshal(ex.RequestHeaders)
	if err != nil {
		return fmt.Errorf("marshal request headers: %w", err)
	}
	respHeaders, err := json.Marshal(ex.ResponseHeaders)
	if err != nil {
		return fmt.Errorf("marshal response headers: %w", err)
	}

	// Truncate bodies that exceed the maximum size
	reqBody := ex.RequestBody
	if len(reqBody) > maxExchangeBodySize {
		reqBody = reqBody[:maxExchangeBodySize]
	}
	respBody := ex.ResponseBody
	if len(respBody) > maxExchangeBodySize {
		respBody = respBody[:maxExchangeBodySize]
	}

	ctx := context.Background()
	err = r.q.SaveExchange(ctx, sqlc.SaveExchangeParams{
		ID:               ex.ID,
		TunnelID:         ex.TunnelID,
		UserID:           userID,
		TraceID:          stringToPgtext(ex.TraceID),
		ReplayRef:        stringToPgtext(ex.ReplayRef),
		Timestamp:        timeToPgtz(ex.Timestamp),
		DurationNs:       int64(ex.Duration),
		Method:           ex.Method,
		Path:             ex.Path,
		Host:             ex.Host,
		RequestHeaders:   reqHeaders,
		RequestBody:      reqBody,
		RequestBodySize:  int32(ex.RequestBodySize),
		ResponseHeaders:  respHeaders,
		ResponseBody:     respBody,
		ResponseBodySize: int32(ex.ResponseBodySize),
		StatusCode:       int32(ex.StatusCode),
		RemoteAddr:       stringToPgtext(ex.RemoteAddr),
	})
	if err != nil {
		return fmt.Errorf("save inspect exchange: %w", err)
	}
	return nil
}

// GetByID retrieves a single exchange by ID. Returns nil, nil if not found.
func (r *ExchangeRepository) GetByID(id string) (*inspect.CapturedExchange, error) {
	ctx := context.Background()
	row, err := r.q.GetExchangeByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get inspect exchange: %w", err)
	}
	return exchangeRowToDomain(
		row.ID, row.TunnelID, row.TraceID, row.ReplayRef,
		row.Timestamp, row.DurationNs,
		row.Method, row.Path, row.Host,
		row.RequestHeaders, row.RequestBody, int64(row.RequestBodySize),
		row.ResponseHeaders, row.ResponseBody, int64(row.ResponseBodySize),
		row.StatusCode, row.RemoteAddr,
	), nil
}

// ListByTunnelID returns exchanges for a tunnel, newest first, with pagination.
func (r *ExchangeRepository) ListByTunnelID(tunnelID string, offset, limit int) ([]*inspect.CapturedExchange, int, error) {
	ctx := context.Background()
	total, err := r.q.CountExchangesByTunnelID(ctx, tunnelID)
	if err != nil {
		return nil, 0, fmt.Errorf("count inspect exchanges: %w", err)
	}

	rows, err := r.q.ListExchangesByTunnelID(ctx, sqlc.ListExchangesByTunnelIDParams{
		TunnelID: tunnelID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list inspect exchanges: %w", err)
	}

	exchanges := make([]*inspect.CapturedExchange, 0, len(rows))
	for _, row := range rows {
		exchanges = append(exchanges, exchangeRowToDomain(
			row.ID, row.TunnelID, row.TraceID, row.ReplayRef,
			row.Timestamp, row.DurationNs,
			row.Method, row.Path, row.Host,
			row.RequestHeaders, row.RequestBody, int64(row.RequestBodySize),
			row.ResponseHeaders, row.ResponseBody, int64(row.ResponseBodySize),
			row.StatusCode, row.RemoteAddr,
		))
	}
	return exchanges, int(total), nil
}

// ListByHostAndUser returns exchanges for a host+user, newest first, with pagination.
func (r *ExchangeRepository) ListByHostAndUser(host string, userID int64, offset, limit int) ([]*inspect.CapturedExchange, int, error) {
	ctx := context.Background()
	total, err := r.q.CountExchangesByHostAndUser(ctx, sqlc.CountExchangesByHostAndUserParams{
		Host:   host,
		UserID: userID,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count inspect exchanges by host: %w", err)
	}

	rows, err := r.q.ListExchangesByHostAndUser(ctx, sqlc.ListExchangesByHostAndUserParams{
		Host:   host,
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list inspect exchanges by host: %w", err)
	}

	exchanges := make([]*inspect.CapturedExchange, 0, len(rows))
	for _, row := range rows {
		exchanges = append(exchanges, exchangeRowToDomain(
			row.ID, row.TunnelID, row.TraceID, row.ReplayRef,
			row.Timestamp, row.DurationNs,
			row.Method, row.Path, row.Host,
			row.RequestHeaders, row.RequestBody, int64(row.RequestBodySize),
			row.ResponseHeaders, row.ResponseBody, int64(row.ResponseBodySize),
			row.StatusCode, row.RemoteAddr,
		))
	}
	return exchanges, int(total), nil
}

// DeleteOlderThan removes exchanges older than the given time.
func (r *ExchangeRepository) DeleteOlderThan(before time.Time) (int64, error) {
	ctx := context.Background()
	count, err := r.q.DeleteExchangesOlderThan(ctx, timeToPgtz(before))
	if err != nil {
		return 0, fmt.Errorf("delete old inspect exchanges: %w", err)
	}
	return count, nil
}

// DeleteByTunnelID removes all exchanges for a tunnel.
func (r *ExchangeRepository) DeleteByTunnelID(tunnelID string) (int64, error) {
	ctx := context.Background()
	count, err := r.q.DeleteExchangesByTunnelID(ctx, tunnelID)
	if err != nil {
		return 0, fmt.Errorf("delete inspect exchanges by tunnel: %w", err)
	}
	return count, nil
}

// exchangeRowToDomain converts sqlc exchange row fields to a domain CapturedExchange.
func exchangeRowToDomain(
	id, tunnelID string,
	traceID, replayRef pgtype.Text,
	timestamp pgtype.Timestamptz,
	durationNs int64,
	method, path, host string,
	reqHeadersJSON, reqBody []byte, reqBodySize int64,
	respHeadersJSON, respBody []byte, respBodySize int64,
	statusCode int32,
	remoteAddr pgtype.Text,
) *inspect.CapturedExchange {
	ex := &inspect.CapturedExchange{
		ID:               id,
		TunnelID:         tunnelID,
		TraceID:          textToString(traceID),
		ReplayRef:        textToString(replayRef),
		Timestamp:        tsToTime(timestamp),
		Duration:         time.Duration(durationNs),
		Method:           method,
		Path:             path,
		Host:             host,
		RequestBody:      reqBody,
		RequestBodySize:  reqBodySize,
		ResponseBody:     respBody,
		ResponseBodySize: respBodySize,
		StatusCode:       int(statusCode),
		RemoteAddr:       textToString(remoteAddr),
	}

	if err := json.Unmarshal(reqHeadersJSON, &ex.RequestHeaders); err != nil {
		ex.RequestHeaders = make(http.Header)
	}
	if err := json.Unmarshal(respHeadersJSON, &ex.ResponseHeaders); err != nil {
		ex.ResponseHeaders = make(http.Header)
	}

	return ex
}
