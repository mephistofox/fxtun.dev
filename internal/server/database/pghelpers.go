package database

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgtype → Go conversions

func tsToTime(ts pgtype.Timestamptz) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{}
}

func tsToTimePtr(ts pgtype.Timestamptz) *time.Time {
	if ts.Valid {
		return &ts.Time
	}
	return nil
}

func textToString(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func textToStringPtr(t pgtype.Text) *string {
	if t.Valid {
		return &t.String
	}
	return nil
}

func int8ToInt64(i pgtype.Int8) int64 {
	if i.Valid {
		return i.Int64
	}
	return 0
}

func int8ToInt64Ptr(i pgtype.Int8) *int64 {
	if i.Valid {
		return &i.Int64
	}
	return nil
}

func int4ToInt(i pgtype.Int4) int {
	if i.Valid {
		return int(i.Int32)
	}
	return 0
}

// Go → pgtype conversions

func timeToPgtz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func timePtrToPgtz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func stringToPgtext(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func stringPtrToPgtext(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func int64ToPgint8(i int64) pgtype.Int8 {
	return pgtype.Int8{Int64: i, Valid: true}
}

func int64PtrToPgint8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

// JSON conversions

func jsonToStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var result []string
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

func stringSliceToJSON(s []string) json.RawMessage {
	if s == nil {
		s = []string{}
	}
	b, _ := json.Marshal(s)
	return b
}

func jsonToMap(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

func mapToJSON(m map[string]interface{}) []byte {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	return b
}

// Error helpers

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
