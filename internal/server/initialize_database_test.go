package server

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makgig/factory/internal/config"
	"go.uber.org/mock/gomock"
)

func TestInitDB_NoDSN(t *testing.T) {
	s := &Server{cfg: &config.ServerConfig{DatabaseDSN: ""}}
	if err := s.initializeDatabase(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestInitDB_OpenError(t *testing.T) {
	s := &Server{
		cfg:         &config.ServerConfig{DatabaseDSN: "dsn"},
		openPool:    func(context.Context, string) (*pgxpool.Pool, error) { return nil, errors.New("open-fail") },
		pingPool:    func(context.Context, *pgxpool.Pool) error { return nil },
		newMigrator: newMigratorDefault,
	}
	if err := s.initializeDatabase(); err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestInitDB_Migrate_NoChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	fake := &pgxpool.Pool{}
	s := &Server{
		cfg:       &config.ServerConfig{DatabaseDSN: "dsn"},
		openPool:  func(context.Context, string) (*pgxpool.Pool, error) { return fake, nil },
		pingPool:  func(context.Context, *pgxpool.Pool) error { return nil },
		closePool: func(*pgxpool.Pool) {},
	}
	m := NewMockMigrator(ctrl)
	m.EXPECT().Up().Return(migrate.ErrNoChange)
	s.newMigrator = func(string) (Migrator, error) { return m, nil }

	if err := s.initializeDatabase(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if s.db != fake {
		t.Fatal("db not set")
	}
}

func TestInitDB_Migrate_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	fake := &pgxpool.Pool{}
	s := &Server{
		cfg:       &config.ServerConfig{DatabaseDSN: "dsn"},
		openPool:  func(context.Context, string) (*pgxpool.Pool, error) { return fake, nil },
		pingPool:  func(context.Context, *pgxpool.Pool) error { return nil },
		closePool: func(*pgxpool.Pool) {},
	}
	m := NewMockMigrator(ctrl)
	m.EXPECT().Up().Return(errors.New("apply-fail"))
	s.newMigrator = func(string) (Migrator, error) { return m, nil }

	if err := s.initializeDatabase(); err == nil {
		t.Fatal("want error, got nil")
	}
	if s.db != nil {
		t.Fatal("db must remain nil")
	}
}
