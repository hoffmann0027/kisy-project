// Package bootstrap creates the initial CEO account. Registration is
// impossible without an invitation and invitations are issued only by
// Level 1, so the very first account must come from configuration.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth/password"
	"kisy-backend/internal/users"
)

// EnsureCEO creates a Level-1 account from env-provided credentials when
// the users table is empty. It is a no-op on any subsequent start.
func EnsureCEO(ctx context.Context, pool *pgxpool.Pool, repo users.Repository, rec audit.Recorder, log *slog.Logger, username, plainPassword string) error {
	if username == "" || plainPassword == "" {
		log.Info("bootstrap: BOOTSTRAP_CEO_USERNAME/PASSWORD not set, skipping")
		return nil
	}

	count, err := repo.Count(ctx, pool)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if len(plainPassword) < 12 {
		return fmt.Errorf("bootstrap: BOOTSTRAP_CEO_PASSWORD must be at least 12 characters")
	}

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return err
	}

	u := &users.User{
		Username:     username,
		DisplayName:  username,
		PasswordHash: hash,
		RoleID:       1, // CEO
		// The seed password comes from configuration and may have been shared
		// out-of-band, so force a change at first login (§2 of the July 2026
		// security update). The frontend blocks the app until it is changed.
		MustChangePassword: true,
	}
	err = repo.Create(ctx, pool, u)
	if errors.Is(err, users.ErrUsernameTaken) {
		// Another instance bootstrapped concurrently; that is fine.
		return nil
	}
	if err != nil {
		return err
	}

	if err := rec.Record(ctx, pool, audit.Event{
		ActorID:    &u.ID,
		Action:     audit.ActionUserBootstrap,
		TargetType: "user",
		TargetID:   &u.ID,
	}); err != nil {
		return err
	}

	log.Info("bootstrap: CEO account created", "username", username)
	return nil
}
