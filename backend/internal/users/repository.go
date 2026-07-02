package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"kisy-backend/internal/platform/db"
)

const pgUniqueViolation = "23505"

// Repository is the persistence port for users. Methods take a db.DBTX so
// they compose into transactions owned by application services.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, u *User) error
	GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, q db.DBTX, username string) (*User, error)
	Count(ctx context.Context, q db.DBTX) (int64, error)
	UpdateUsername(ctx context.Context, q db.DBTX, id uuid.UUID, username string) error
	UpdatePasswordHash(ctx context.Context, q db.DBTX, id uuid.UUID, hash string) error
	// AdminResetPasswordHash sets a new hash and forces a change on next
	// login (used when the CEO resets someone's credentials).
	AdminResetPasswordHash(ctx context.Context, q db.DBTX, id uuid.UUID, hash string) error
	UpdateRole(ctx context.Context, q db.DBTX, id uuid.UUID, roleID int) error
	SetActive(ctx context.Context, q db.DBTX, id uuid.UUID, active bool) error
	List(ctx context.Context, q db.DBTX, limit, offset int) ([]User, error)
	// RegisterLoginFailure atomically increments the failure counter and,
	// when maxAttempts is reached, sets locked_until to lockUntil.
	// Returns the resulting lock timestamp (nil if not locked).
	RegisterLoginFailure(ctx context.Context, q db.DBTX, id uuid.UUID, maxAttempts int, lockUntil time.Time) (*time.Time, error)
	ResetLoginFailures(ctx context.Context, q db.DBTX, id uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const userColumns = `
	id, username::text, display_name, password_hash, role_id, avatar_url,
	status, last_seen_at, is_active, failed_login_attempts, locked_until,
	must_change_password, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.RoleID, &u.AvatarURL,
		&u.Status, &u.LastSeenAt, &u.IsActive, &u.FailedLoginAttempts, &u.LockedUntil,
		&u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("users: scan: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, u *User) error {
	err := q.QueryRow(ctx, `
		INSERT INTO users (username, display_name, password_hash, role_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, is_active, failed_login_attempts, must_change_password, created_at, updated_at`,
		u.Username, u.DisplayName, u.PasswordHash, u.RoleID,
	).Scan(&u.ID, &u.Status, &u.IsActive, &u.FailedLoginAttempts, &u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return ErrUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("users: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*User, error) {
	return scanUser(q.QueryRow(ctx, `SELECT`+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *PostgresRepository) GetByUsername(ctx context.Context, q db.DBTX, username string) (*User, error) {
	return scanUser(q.QueryRow(ctx, `SELECT`+userColumns+` FROM users WHERE username = $1`, username))
}

func (r *PostgresRepository) Count(ctx context.Context, q db.DBTX) (int64, error) {
	var n int64
	if err := q.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("users: count: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) UpdateUsername(ctx context.Context, q db.DBTX, id uuid.UUID, username string) error {
	tag, err := q.Exec(ctx, `UPDATE users SET username = $2 WHERE id = $1`, id, username)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return ErrUsernameTaken
	}
	if err != nil {
		return fmt.Errorf("users: update username: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdatePasswordHash(ctx context.Context, q db.DBTX, id uuid.UUID, hash string) error {
	tag, err := q.Exec(ctx, `UPDATE users SET password_hash = $2, must_change_password = false WHERE id = $1`, id, hash)
	if err != nil {
		return fmt.Errorf("users: update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) AdminResetPasswordHash(ctx context.Context, q db.DBTX, id uuid.UUID, hash string) error {
	tag, err := q.Exec(ctx, `
		UPDATE users SET password_hash = $2, must_change_password = true,
			failed_login_attempts = 0, locked_until = NULL
		WHERE id = $1`, id, hash)
	if err != nil {
		return fmt.Errorf("users: admin reset password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateRole(ctx context.Context, q db.DBTX, id uuid.UUID, roleID int) error {
	tag, err := q.Exec(ctx, `UPDATE users SET role_id = $2 WHERE id = $1`, id, roleID)
	if err != nil {
		return fmt.Errorf("users: update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetActive(ctx context.Context, q db.DBTX, id uuid.UUID, active bool) error {
	tag, err := q.Exec(ctx, `UPDATE users SET is_active = $2 WHERE id = $1`, id, active)
	if err != nil {
		return fmt.Errorf("users: set active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, q db.DBTX, limit, offset int) ([]User, error) {
	rows, err := q.Query(ctx, `SELECT`+userColumns+`
		FROM users ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("users: list: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.RoleID, &u.AvatarURL,
			&u.Status, &u.LastSeenAt, &u.IsActive, &u.FailedLoginAttempts, &u.LockedUntil,
			&u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("users: scan list row: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) RegisterLoginFailure(ctx context.Context, q db.DBTX, id uuid.UUID, maxAttempts int, lockUntil time.Time) (*time.Time, error) {
	var lockedUntil *time.Time
	err := q.QueryRow(ctx, `
		UPDATE users SET
			failed_login_attempts = failed_login_attempts + 1,
			locked_until = CASE
				WHEN failed_login_attempts + 1 >= $2 THEN $3
				ELSE locked_until
			END
		WHERE id = $1
		RETURNING locked_until`,
		id, maxAttempts, lockUntil,
	).Scan(&lockedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("users: register login failure: %w", err)
	}
	return lockedUntil, nil
}

func (r *PostgresRepository) ResetLoginFailures(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	if _, err := q.Exec(ctx, `
		UPDATE users SET failed_login_attempts = 0, locked_until = NULL WHERE id = $1`, id,
	); err != nil {
		return fmt.Errorf("users: reset login failures: %w", err)
	}
	return nil
}
