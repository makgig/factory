package pgerrors

import (
	"errors"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// IsRetriable сообщает, что ошибка — транспортная (класс 08*, Connection Exception)
// и её можно повторить. Пример явной non-retriable — UniqueViolation.
func IsRetriable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	// Вся группа Class 08 — Connection Exception — Retriable.
	if strings.HasPrefix(pgErr.Code, "08") {
		return true
	}
	// Пример явной non-retriable — нарушение уникальности.
	switch pgErr.Code {
	case pgerrcode.UniqueViolation:
		return false
	}
	return false
}
