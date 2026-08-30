// Command resetpw sets a user's password directly against the database.
//
// It exists because the product has no other way out of one situation: the
// CEO is the only role that may reset passwords, so if the sole Level-1
// account loses its password nobody can restore it. Registration needs an
// invite, invites need a CEO, and the bootstrap that seeds the first account
// only runs while the users table is empty — on a live database it is a
// no-op. Without this the answer would be "wipe the database".
//
// It is deliberately an operator tool, not an endpoint: it needs direct
// database credentials, which only whoever runs the deployment has.
//
//	DATABASE_URL=postgres://… go run ./cmd/resetpw -user ceo
//	DATABASE_URL=postgres://… go run ./cmd/resetpw -user ceo -password '…'
//	go run ./cmd/resetpw -password '…' -hash-only     # no database needed
//
// The new password must be changed at the next sign-in: an operator-chosen
// secret has travelled through a shell and a terminal history, so it is
// treated as compromised from the start — the same rule the bootstrap
// account follows.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/auth/password"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "resetpw:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		username = flag.String("user", "", "username whose password to reset")
		plain    = flag.String("password", "", "new password; generated when empty")
		hashOnly = flag.Bool("hash-only", false, "print the hash and exit, touching no database")
	)
	flag.Parse()

	newPassword := *plain
	generated := false
	if newPassword == "" {
		var err error
		if newPassword, err = generatePassword(); err != nil {
			return err
		}
		generated = true
	}
	if len(newPassword) < 12 {
		return errors.New("password must be at least 12 characters")
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// hash-only keeps the tool usable where the database is only reachable
	// from a web console: paste the UPDATE there instead.
	if *hashOnly {
		fmt.Printf("password: %s\nhash:     %s\n\n", newPassword, hash)
		fmt.Printf("UPDATE users SET password_hash = '%s', must_change_password = true,\n"+
			"       failed_login_attempts = 0, locked_until = NULL\n"+
			" WHERE username = '%s';\n", hash, *username)
		return nil
	}

	if *username == "" {
		return errors.New("-user is required (or use -hash-only)")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clearing the lockout matters as much as the hash: a forgotten password
	// is usually discovered by guessing at it until the account locks.
	tag, err := tx.Exec(ctx, `
		UPDATE users
		   SET password_hash = $2,
		       must_change_password = true,
		       failed_login_attempts = 0,
		       locked_until = NULL
		 WHERE username = $1`, *username, hash)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("no user named %q", *username)
	}

	// Any session opened with the old credentials is now suspect — whoever
	// forced the reset should be the only one holding a live session.
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = now()
		 WHERE user_id = (SELECT id FROM users WHERE username = $1)
		   AND revoked_at IS NULL`, *username); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	fmt.Printf("password reset for %q\n", *username)
	if generated {
		fmt.Printf("new password: %s\n", newPassword)
	}
	fmt.Println("all sessions revoked; the password must be changed at next sign-in")
	return nil
}

// generatePassword returns 24 URL-safe characters from crypto/rand — enough
// entropy that the value surviving in a shell history is the only real risk.
func generatePassword() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
