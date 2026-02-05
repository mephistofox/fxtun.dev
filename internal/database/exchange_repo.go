package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mephistofox/fxtunnel/internal/inspect"
)

// ExchangeRepository handles inspect exchange persistence
type ExchangeRepository struct {
	db *sql.DB
}

// NewExchangeRepository creates a new exchange repository
func NewExchangeRepository(db *sql.DB) *ExchangeRepository {
	return &ExchangeRepository{db: db}
}

// Save persists a captured exchange to the database
func (r *ExchangeRepository) Save(ex *inspect.CapturedExchange, userID int64) error {
	reqHeaders, err := json.Marshal(ex.RequestHeaders)
	if err != nil {
		return fmt.Errorf("marshal request headers: %w", err)
	}
	respHeaders, err := json.Marshal(ex.ResponseHeaders)
	if err != nil {
		return fmt.Errorf("marshal response headers: %w", err)
	}

	query := `
		INSERT INTO inspect_exchanges (
			id, tunnel_id, user_id, trace_id, replay_ref, timestamp, duration_ns,
			method, path, host, request_headers, request_body, request_body_size,
			response_headers, response_body, response_body_size, status_code, remote_addr
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(query,
		ex.ID, ex.TunnelID, userID, ex.TraceID, ex.ReplayRef,
		ex.Timestamp, int64(ex.Duration),
		ex.Method, ex.Path, ex.Host,
		string(reqHeaders), ex.RequestBody, ex.RequestBodySize,
		string(respHeaders), ex.ResponseBody, ex.ResponseBodySize,
		ex.StatusCode, ex.RemoteAddr,
	)
	if err != nil {
		return fmt.Errorf("save inspect exchange: %w", err)
	}
	return nil
}

// GetByID retrieves a single exchange by ID
func (r *ExchangeRepository) GetByID(id string) (*inspect.CapturedExchange, error) {
	query := `
		SELECT id, tunnel_id, trace_id, replay_ref, timestamp, duration_ns,
			method, path, host, request_headers, request_body, request_body_size,
			response_headers, response_body, response_body_size, status_code, remote_addr
		FROM inspect_exchanges WHERE id = ?
	`

	ex := &inspect.CapturedExchange{}
	var traceID, replayRef, remoteAddr sql.NullString
	var reqHeadersJSON, respHeadersJSON string
	var durationNs int64

	err := r.db.QueryRow(query, id).Scan(
		&ex.ID, &ex.TunnelID, &traceID, &replayRef,
		&ex.Timestamp, &durationNs,
		&ex.Method, &ex.Path, &ex.Host,
		&reqHeadersJSON, &ex.RequestBody, &ex.RequestBodySize,
		&respHeadersJSON, &ex.ResponseBody, &ex.ResponseBodySize,
		&ex.StatusCode, &remoteAddr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get inspect exchange: %w", err)
	}

	ex.Duration = time.Duration(durationNs)
	if traceID.Valid {
		ex.TraceID = traceID.String
	}
	if replayRef.Valid {
		ex.ReplayRef = replayRef.String
	}
	if remoteAddr.Valid {
		ex.RemoteAddr = remoteAddr.String
	}

	if err := json.Unmarshal([]byte(reqHeadersJSON), &ex.RequestHeaders); err != nil {
		ex.RequestHeaders = make(http.Header)
	}
	if err := json.Unmarshal([]byte(respHeadersJSON), &ex.ResponseHeaders); err != nil {
		ex.ResponseHeaders = make(http.Header)
	}

	return ex, nil
}

// ListByTunnelID returns exchanges for a tunnel, newest first, with pagination
func (r *ExchangeRepository) ListByTunnelID(tunnelID string, offset, limit int) ([]*inspect.CapturedExchange, int, error) {
	// Get total count
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM inspect_exchanges WHERE tunnel_id = ?", tunnelID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count inspect exchanges: %w", err)
	}

	query := `
		SELECT id, tunnel_id, trace_id, replay_ref, timestamp, duration_ns,
			method, path, host, request_headers, request_body, request_body_size,
			response_headers, response_body, response_body_size, status_code, remote_addr
		FROM inspect_exchanges WHERE tunnel_id = ?
		ORDER BY timestamp DESC LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, tunnelID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list inspect exchanges: %w", err)
	}
	defer rows.Close()

	var exchanges []*inspect.CapturedExchange
	for rows.Next() {
		ex := &inspect.CapturedExchange{}
		var traceID, replayRef, remoteAddr sql.NullString
		var reqHeadersJSON, respHeadersJSON string
		var durationNs int64

		if err := rows.Scan(
			&ex.ID, &ex.TunnelID, &traceID, &replayRef,
			&ex.Timestamp, &durationNs,
			&ex.Method, &ex.Path, &ex.Host,
			&reqHeadersJSON, &ex.RequestBody, &ex.RequestBodySize,
			&respHeadersJSON, &ex.ResponseBody, &ex.ResponseBodySize,
			&ex.StatusCode, &remoteAddr,
		); err != nil {
			return nil, 0, fmt.Errorf("scan inspect exchange: %w", err)
		}

		ex.Duration = time.Duration(durationNs)
		if traceID.Valid {
			ex.TraceID = traceID.String
		}
		if replayRef.Valid {
			ex.ReplayRef = replayRef.String
		}
		if remoteAddr.Valid {
			ex.RemoteAddr = remoteAddr.String
		}

		if err := json.Unmarshal([]byte(reqHeadersJSON), &ex.RequestHeaders); err != nil {
			ex.RequestHeaders = make(http.Header)
		}
		if err := json.Unmarshal([]byte(respHeadersJSON), &ex.ResponseHeaders); err != nil {
			ex.ResponseHeaders = make(http.Header)
		}

		exchanges = append(exchanges, ex)
	}

	return exchanges, total, nil
}

// DeleteOlderThan removes exchanges older than the given time
func (r *ExchangeRepository) DeleteOlderThan(before time.Time) (int64, error) {
	result, err := r.db.Exec("DELETE FROM inspect_exchanges WHERE created_at < ?", before)
	if err != nil {
		return 0, fmt.Errorf("delete old inspect exchanges: %w", err)
	}
	count, _ := result.RowsAffected()
	return count, nil
}

// DeleteByTunnelID removes all exchanges for a tunnel
func (r *ExchangeRepository) DeleteByTunnelID(tunnelID string) (int64, error) {
	result, err := r.db.Exec("DELETE FROM inspect_exchanges WHERE tunnel_id = ?", tunnelID)
	if err != nil {
		return 0, fmt.Errorf("delete inspect exchanges by tunnel: %w", err)
	}
	count, _ := result.RowsAffected()
	return count, nil
}
