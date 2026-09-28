//go:build integration

package auth_test

import (
	"context"
	"errors"
	"testing"

	"kisy-backend/internal/auth"
	"kisy-backend/internal/users"
)

// A login that belonged to a deleted account is never handed to anyone else
// (audit E-01): mentions, links and other people's memory still name it.
func TestRegistrationRefusesTheLoginOfADeletedAccount(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	if _, err := e.pool.Exec(ctx, `INSERT INTO retired_usernames (username) VALUES ('gone_person')`); err != nil {
		t.Fatal(err)
	}

	_, err := e.svc.Register(ctx, "", "gone_person", "Новый Человек", "long-enough-pass-1", auth.ClientMeta{})
	if !errors.Is(err, users.ErrUsernameTaken) {
		t.Fatalf("registering a retired login: %v, want ErrUsernameTaken", err)
	}

	// Capitalisation is not a way around it (the column is CITEXT).
	if _, err := e.svc.Register(ctx, "", "Gone_Person", "Новый Человек", "long-enough-pass-1", auth.ClientMeta{}); !errors.Is(err, users.ErrUsernameTaken) {
		t.Fatalf("registering it in another case: %v, want ErrUsernameTaken", err)
	}

	// An untouched login still works.
	if _, err := e.svc.Register(ctx, "", "fresh_person", "Другой Человек", "long-enough-pass-1", auth.ClientMeta{}); err != nil {
		t.Fatalf("an ordinary sign-up: %v", err)
	}
}
