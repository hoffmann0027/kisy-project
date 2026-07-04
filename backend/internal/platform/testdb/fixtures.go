package testdb

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedUser inserts an active user at the given clearance level and returns
// its id. The password hash is a placeholder — service-layer integration
// tests operate below the login flow.
func SeedUser(t *testing.T, pool *pgxpool.Pool, username string, roleLevel int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (username, display_name, password_hash, role_id)
		VALUES ($1, $2, 'x-placeholder-hash', $3)
		RETURNING id`, username, username, roleLevel).Scan(&id)
	if err != nil {
		t.Fatalf("testdb: seed user %q: %v", username, err)
	}
	return id
}
