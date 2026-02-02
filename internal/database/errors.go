package database

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrPlanNotFound = errors.New("plan not found")
	ErrPlanHasUsers = errors.New("plan has users assigned")
)

// notFoundOrError returns the sentinel error if the underlying error is
// sql.ErrNoRows, otherwise wraps the error with the given context string.
func notFoundOrError(err error, sentinel error, context string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return sentinel
	}
	return fmt.Errorf("%s: %w", context, err)
}
