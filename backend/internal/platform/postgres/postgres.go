// Package postgres wires the connection pool and schema migrations.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a connection pool from a DSN and verifies connectivity
// with a ping, so startup fails fast if the database is unreachable rather
// than surfacing the error on the first request.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return pool, nil
}

// Migrate applies all pending migrations found in migrationsPath. The dsn
// may use the postgres:// or postgresql:// scheme (managed providers use
// either); it is translated to the pgx5 driver scheme that golang-migrate's
// pgx v5 database driver expects.
func Migrate(dsn, migrationsPath string) error {
	trimmed := strings.TrimPrefix(dsn, "postgresql://")
	trimmed = strings.TrimPrefix(trimmed, "postgres://")
	migrateDSN := "pgx5://" + trimmed

	m, err := migrate.New("file://"+migrationsPath, migrateDSN)
	if err != nil {
		return fmt.Errorf("postgres: init migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: apply migrations: %w", err)
	}

	return nil
}
