package server

import (
	"context"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockgen -destination=mock_migrator_test.go -package=server . Migrator

// Единственный интерфейс, который будем мокать через gomock.
type Migrator interface {
	Up() error
}

// Функции-зависимости, которые легко подменять в тестах.
type openPoolFunc func(ctx context.Context, dsn string) (*pgxpool.Pool, error)
type pingPoolFunc func(ctx context.Context, p *pgxpool.Pool) error
type newMigratorFunc func(dsn string) (Migrator, error)

// Прод-реализации:
var (
	openPoolDefault openPoolFunc = func(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
		return pgxpool.New(ctx, dsn)
	}
	pingPoolDefault pingPoolFunc = func(ctx context.Context, p *pgxpool.Pool) error {
		return p.Ping(ctx)
	}
	newMigratorDefault newMigratorFunc = func(dsn string) (Migrator, error) {
		return migrate.New("file://migrations", dsn)
	}
)
