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
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/postgres"
)

// New creates a fresh migrated database and returns a pool connected to it.
// The database is dropped on test cleanup. Without TEST_DATABASE_URL the
// suite is skipped on a developer's machine and FAILS in CI — see
// missingURLIsFatal.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	adminURL := AdminURL(t)

	ctx := context.Background()
	// #nosec G404 -- non-security use: a unique throwaway test database name.
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

// AdminURL returns TEST_DATABASE_URL, or skips the test on a developer's
// machine and FAILS it in CI when the variable is unset. Every integration
// harness goes through it: auth's own setup read the variable directly and
// skipped, so it escaped the CI rule below (audit D-12).
func AdminURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if missingURLIsFatal(os.Getenv("CI")) {
			t.Fatal("testdb: TEST_DATABASE_URL is not set in CI — the integration suite would report success without running")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	return url
}

// missingURLIsFatal decides what an unset TEST_DATABASE_URL means, from the
// value of the CI environment variable (GitHub Actions sets CI=true).
//
// On a developer's machine skipping is right: `go test ./...` should stay
// green without a database. In CI it is the opposite — a skip there means the
// whole integration suite quietly reported success without running a single
// query, which is how the HIGH findings of the September audit lived in code
// that "had tests" (audit D-12).
func missingURLIsFatal(ci string) bool {
	switch strings.ToLower(strings.TrimSpace(ci)) {
	case "", "0", "false":
		return false
	default:
		return true
	}
}
