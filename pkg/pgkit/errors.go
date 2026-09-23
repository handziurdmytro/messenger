package pgkit

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	CodeNotNullViolation    = "23502"
	CodeForeignKeyViolation = "23503"
	CodeUniqueViolation     = "23505"
	CodeCheckViolation      = "23514"

	CodeSerializationFailure = "40001"
	CodeDeadlockDetected     = "40P01"
)

func ErrorCode(err error) (string, bool) {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return "", false
	}

	return pgErr.Code, true
}

func ConstraintName(err error) (string, bool) {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return "", false
	}

	if pgErr.ConstraintName == "" {
		return "", false
	}

	return pgErr.ConstraintName, true
}

func IsCode(err error, code string) bool {
	got, ok := ErrorCode(err)
	return ok && got == code
}

func IsConstraint(
	err error,
	code string,
	constraint string,
) bool {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == code &&
		pgErr.ConstraintName == constraint
}
