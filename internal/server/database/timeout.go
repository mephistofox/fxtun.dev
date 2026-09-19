package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// queryTimeout bounds a single database round-trip.
//
// Every repository calls the generated queries with context.Background(), so
// without this nothing ever gives up: a frozen or unreachable Postgres leaves
// each in-flight request blocked forever, holding its goroutine and its client
// socket, and once the pool is drained every later request blocks in Acquire
// as well. The deadline also bounds Acquire, so a stalled pool drains instead
// of growing without limit.
//
// It is deliberately generous — hourly retention deletes run through the same
// path as a login.
const queryTimeout = 30 * time.Second

// timeoutDB wraps a pgxpool so every generated query carries a deadline.
// It implements sqlc.DBTX.
type timeoutDB struct {
	pool *pgxpool.Pool
}

func (d timeoutDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	return d.pool.Exec(ctx, sql, args...)
}

// Query keeps the deadline alive until the caller closes the rows, because
// pgx streams them lazily. The generated code always defers rows.Close().
func (d timeoutDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		cancel()
		return nil, err
	}
	return &timeoutRows{Rows: rows, cancel: cancel}, nil
}

// QueryRow holds the deadline until Scan, which is where pgx actually reads.
func (d timeoutDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	return timeoutRow{row: d.pool.QueryRow(ctx, sql, args...), cancel: cancel}
}

type timeoutRows struct {
	pgx.Rows
	cancel context.CancelFunc
}

func (r *timeoutRows) Close() {
	r.Rows.Close()
	r.cancel()
}

type timeoutRow struct {
	row    pgx.Row
	cancel context.CancelFunc
}

func (r timeoutRow) Scan(dest ...any) error {
	defer r.cancel()
	return r.row.Scan(dest...)
}
