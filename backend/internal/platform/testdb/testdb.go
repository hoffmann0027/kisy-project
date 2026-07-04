// Package testdb provides a shared harness for integration tests: it spins
// up a uniquely-named, freshly-migrated database from the maintenance URL
// in TEST_DATABASE_URL and tears it down afterwards. Tests skip when the
// variable is unset, so `go test ./...` stays green without a database.
//
// It is imported only from _test.go files (behind the `integration` build
// tag) but is a normal package so it can be shared across module tests.
package testdb

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/postgres"
)

// New creates a fresh migrated database and returns a pool connected to it.
// The database is dropped on test cleanup. The suite is skipped when
// TEST_DATABASE_URL is not set.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	dbName := fmt.Sprintf("kisy_it_%d", rand.Int63n(1_000_000_000))

	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("testdb: connect admin: %v", err)
	}
	defer admin.Close()

	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, dbName)); err != nil {
		t.Fatalf("testdb: create database: %v", err)
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("testdb: parse url: %v", err)
	}
	u.Path = "/" + dbName
	testURL := u.String()

	if err := postgres.Migrate(testURL, migrationsDir(t)); err != nil {
		t.Fatalf("testdb: migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, testURL)
	if err != nil {
		t.Fatalf("testdb: connect test db: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanup, err := pgxpool.New(context.Background(), adminURL)
		if err != nil {
			return
		}
		defer cleanup.Close()
		_, _ = cleanup.Exec(context.Background(), fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, dbName))
	})

	return pool
}

// migrationsDir walks up from the test's working directory until it finds
// the backend/migrations directory, so tests work regardless of which
// package they live in.
func migrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("testdb: getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if _, err := os.Stat(filepath.Join(candidate, "000001_enable_extensions.up.sql")); err == nil {
				return filepath.ToSlash(candidate)
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("testdb: could not locate migrations directory")
	return ""
}
