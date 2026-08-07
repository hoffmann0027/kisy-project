package config

import (
	"strings"
	"testing"
)

// setBaseEnv provides the minimum viable configuration; individual tests
// override what they probe. t.Setenv auto-restores after each test.
func setBaseEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"APP_ENV":            "development",
		"POSTGRES_HOST":      "postgres",
		"POSTGRES_PASSWORD":  "test-db-password",
		"POSTGRES_SSLMODE":   "disable",
		"REDIS_HOST":         "redis",
		"REDIS_PASSWORD":     "test-redis-password",
		"JWT_ACCESS_SECRET":  strings.Repeat("a", 32),
		"JWT_REFRESH_SECRET": strings.Repeat("b", 32),
		"WS_ALLOWED_ORIGIN":  "https://kisy.example",
		"DATABASE_URL":       "",
		"REDIS_URL":          "",
		"ALLOW_PLAINTEXT_DB": "",
		"IP_HASH_SALT":       "",
		"TURN_SECRET":        "",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadDevelopmentPermissive(t *testing.T) {
	setBaseEnv(t)
	// Development tolerates what production must refuse.
	t.Setenv("WS_ALLOWED_ORIGIN", "")
	t.Setenv("REDIS_PASSWORD", "")
	if _, err := Load(); err != nil {
		t.Fatalf("development load: %v", err)
	}
}

func TestProductionHappyPath(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_SSLMODE", "require")
	if _, err := Load(); err != nil {
		t.Fatalf("production load: %v", err)
	}
}

func TestProductionRequiresWSOrigin(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("WS_ALLOWED_ORIGIN", "")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "WS_ALLOWED_ORIGIN") {
		t.Fatalf("want WS_ALLOWED_ORIGIN error, got %v", err)
	}
}

func TestProductionSameOriginDeploySkipsWSOrigin(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("WS_ALLOWED_ORIGIN", "")
	// A single-service deploy that serves the SPA itself (WEB_DIR set) is
	// same-origin, so an explicit WS_ALLOWED_ORIGIN is not required.
	t.Setenv("WEB_DIR", "/app/web")
	if _, err := Load(); err != nil {
		t.Fatalf("same-origin deploy should not require WS_ALLOWED_ORIGIN: %v", err)
	}
}

func TestProductionRejectsPlaintextDBOverNetwork(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	// sslmode=disable to the docker-network host "postgres" must fail…
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_SSLMODE") {
		t.Fatalf("want sslmode error, got %v", err)
	}
	// …unless the operator opts in for a single-host private network…
	t.Setenv("ALLOW_PLAINTEXT_DB", "true")
	if _, err := Load(); err != nil {
		t.Fatalf("opt-in should pass: %v", err)
	}
	// …and loopback (all-in-one image) is fine without the opt-in.
	t.Setenv("ALLOW_PLAINTEXT_DB", "")
	t.Setenv("POSTGRES_HOST", "localhost")
	if _, err := Load(); err != nil {
		t.Fatalf("loopback should pass: %v", err)
	}
}

func TestProductionRejectsPlaintextDatabaseURL(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://kisy:pw@db.internal:5432/kisy?sslmode=disable")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_SSLMODE") {
		t.Fatalf("want sslmode error for DATABASE_URL, got %v", err)
	}
	// Managed URL without explicit sslmode=disable passes (driver negotiates TLS).
	t.Setenv("DATABASE_URL", "postgres://kisy:pw@db.internal:5432/kisy")
	if _, err := Load(); err != nil {
		t.Fatalf("managed URL should pass: %v", err)
	}
}

func TestProductionRequiresRedisPassword(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("REDIS_PASSWORD", "")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "REDIS_PASSWORD") {
		t.Fatalf("want REDIS_PASSWORD error, got %v", err)
	}
}

func TestProductionRejectsBurnedSecrets(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_SSLMODE", "require")

	// Any value whose digest is on the compromised list must be refused.
	// We can't use the real burned values in a test, so instead assert the
	// mechanism via a synthetic entry.
	burned := "unit-test-burned-secret-value-123456"
	knownCompromisedDigests[sha256Hex(burned)] = "JWT_ACCESS_SECRET (unit test)"
	defer delete(knownCompromisedDigests, sha256Hex(burned))

	t.Setenv("JWT_ACCESS_SECRET", burned)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "known-compromised") {
		t.Fatalf("want known-compromised error, got %v", err)
	}
}

func TestDBPoolDefaults(t *testing.T) {
	setBaseEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Explicit defaults matter: pgx would otherwise size the pool from the
	// CPU count, which can exhaust a managed Postgres once replicas multiply.
	if cfg.DBPool.MaxConns != 10 || cfg.DBPool.MinConns != 2 {
		t.Fatalf("unexpected pool sizing: max=%d min=%d", cfg.DBPool.MaxConns, cfg.DBPool.MinConns)
	}
	if cfg.DBPool.MaxConnIdleTime == 0 || cfg.DBPool.MaxConnLifetime == 0 || cfg.DBPool.HealthCheckPeriod == 0 {
		t.Fatal("pool lifetimes must have non-zero defaults")
	}
}

func TestDBPoolOverrides(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DB_POOL_MAX_CONNS", "25")
	t.Setenv("DB_POOL_MIN_CONNS", "5")
	t.Setenv("DB_POOL_MAX_CONN_IDLE_TIME", "90s")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.DBPool.MaxConns != 25 || cfg.DBPool.MinConns != 5 {
		t.Fatalf("overrides ignored: max=%d min=%d", cfg.DBPool.MaxConns, cfg.DBPool.MinConns)
	}
	if cfg.DBPool.MaxConnIdleTime.String() != "1m30s" {
		t.Fatalf("idle override ignored: %s", cfg.DBPool.MaxConnIdleTime)
	}
}

func TestDBPoolRejectsInvalidSizing(t *testing.T) {
	// MinConns above MaxConns would make pgx fail deep inside pool creation;
	// catch it at config load with a clear message instead.
	setBaseEnv(t)
	t.Setenv("DB_POOL_MAX_CONNS", "4")
	t.Setenv("DB_POOL_MIN_CONNS", "9")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when MinConns exceeds MaxConns")
	}

	setBaseEnv(t)
	t.Setenv("DB_POOL_MAX_CONNS", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when MaxConns is zero")
	}
}
