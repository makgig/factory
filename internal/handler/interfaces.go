package handler

import "context"

//go:generate mockgen -destination=mock_dbpinger_test.go -package=handler . DBPinger

// DBPinger — минимальный контракт для health-check.
type DBPinger interface {
	Ping(ctx context.Context) error
}
