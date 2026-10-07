// Package postgres wires the connection pool and schema migrations.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/dsn"
)

// PoolSettings bounds the connection pool. Left at their zero values, pgx
// applies its own defaults (MaxConns = max(4, NumCPU), no idle timeout), which
// is why the server passes explicit values from configuration — an unbounded
// pool per replica can exhaust a managed Postgres' connection limit.
type PoolSettings struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// NewPool creates a connection pool from a DSN and verifies connectivity
// with a ping, so startup fails fast if the database is unreachable rather
// than surfacing the error on the first request. Zero-valued settings fall
// back to the pgx defaults, so callers may pass a partially filled struct.
// No error carries the DSN's password (audit A-39).
func NewPool(ctx context.Context, connString string, s PoolSettings) (*pgxpool.Pool, error) {
	pool, err := newPool(ctx, connString, s)
	return pool, dsn.Scrub(err, connString)
}

func newPool(ctx context.Context, connString string, s PoolSettings) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	if s.MaxConns > 0 {
		cfg.MaxConns = s.MaxConns
	}
	if s.MinConns > 0 {
		cfg.MinConns = s.MinConns
	}
	if s.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = s.MaxConnLifetime
	}
	if s.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = s.MaxConnIdleTime
	}
	if s.HealthCheckPeriod > 0 {
		cfg.HealthCheckPeriod = s.HealthCheckPeriod
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
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
// pgx v5 database driver expects. golang-migrate quotes a URL it cannot
// parse in full, so the error is scrubbed of the password (audit A-39).
func Migrate(connString, migrationsPath string) error {
	trimmed := strings.TrimPrefix(connString, "postgresql://")
	trimmed = strings.TrimPrefix(trimmed, "postgres://")
	migrateDSN := "pgx5://" + trimmed
	return dsn.Scrub(migrateUp(migrateDSN, migrationsPath), migrateDSN)
}

func migrateUp(migrateDSN, migrationsPath string) error {
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
