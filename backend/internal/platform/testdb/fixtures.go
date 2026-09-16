package testdb

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedDisplayName derives a display name that passes the rule from a test
// username: display names are letters only and unique (migration 46), while
// test usernames like "emp_2" are not. Digits become letters and underscores
// go, so distinct usernames stay distinct names in practice.
func SeedDisplayName(username string) string {
	out := make([]rune, 0, len(username))
	for _, r := range username {
		switch {
		case r >= '0' && r <= '9':
			out = append(out, 'a'+(r-'0'))
		case r == '_':
		default:
			out = append(out, r)
		}
	}
	if len(out) < 2 {
		out = append(out, 'x', 'x')
	}
	return string(out)
}

// SeedUser inserts an active user at the given clearance level and returns
// its id. The password hash is a placeholder — service-layer integration
// tests operate below the login flow.
// A roleLevel of access.NoLevel (0) seeds a basic account — one registered
// without an invitation, which has no level at all.
func SeedUser(t *testing.T, pool *pgxpool.Pool, username string, roleLevel int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (username, display_name, password_hash, role_id, account_kind)
		VALUES ($1, $2, 'x-placeholder-hash', NULLIF($3, 0),
		        CASE WHEN $3 = 0 THEN 'basic' ELSE 'invited' END)
		RETURNING id`, username, SeedDisplayName(username), roleLevel).Scan(&id)
	if err != nil {
		t.Fatalf("testdb: seed user %q: %v", username, err)
	}
	return id
}
