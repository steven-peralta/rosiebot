package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func NewMigrator(dsn string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("postgres: load migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL(dsn))
	if err != nil {
		return nil, fmt.Errorf("postgres: init migrator: %w", err)
	}
	return m, nil
}

func Migrate(dsn string) error {
	m, err := NewMigrator(dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: migrate up: %w", err)
	}
	return nil
}

func migrateURL(dsn string) string {
	for _, scheme := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(dsn, scheme) {
			return "pgx5://" + strings.TrimPrefix(dsn, scheme)
		}
	}
	return dsn
}

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}
