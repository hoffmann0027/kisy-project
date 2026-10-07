package postgres

import (
	"context"
	"strings"
	"testing"
)

// Audit A-39: a DATABASE_URL with a typo put the password in the log —
// `parse "pgx5://kisy:SuperSecretPW1@host:notaport/kisy": invalid port`.
func TestConnectionErrorsDoNotCarryThePassword(t *testing.T) {
	const password = "SuperSecretPW1"
	bad := "postgres://kisy:" + password + "@host:notaport/kisy"

	// The real migrations: the source opens, the database URL is what fails.
	err := Migrate(bad, "../../../migrations")
	if err == nil || strings.Contains(err.Error(), password) {
		t.Fatalf("migrate: %v", err)
	}
	_, err = NewPool(context.Background(), bad, PoolSettings{})
	if err == nil || strings.Contains(err.Error(), password) {
		t.Fatalf("pool: %v", err)
	}
}
