package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"kisy-backend/internal/access"
	"kisy-backend/internal/platform/db"
)

const pgUniqueViolation = "23505"

// The unique index on the display-name key (migration 46). Named so a
// collision on the name is not reported as a taken username.
const displayNameKeyIndex = "uq_users_display_name_key"

// uniqueViolation turns a unique-index failure into the domain error for the
// column that collided, or returns nil for any other error (or none).
func uniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgUniqueViolation {
		return nil
	}
	if strings.HasPrefix(pgErr.ConstraintName, displayNameKeyIndex) {
		return ErrDisplayNameTaken
	}
	return ErrUsernameTaken
}

// Repository is the persistence port for users. Methods take a db.DBTX so
// they compose into transactions owned by application services.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, u *User) error
	GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, q db.DBTX, username string) (*User, error)
	Count(ctx context.Context, q db.DBTX) (int64, error)
	UpdateUsername(ctx context.Context, q db.DBTX, id uuid.UUID, username string) error
	// UpdateDisplayName changes the human-facing name shown across the UI.
	UpdateDisplayName(ctx context.Context, q db.DBTX, id uuid.UUID, displayName string) error
	// SetAvatarURL points the user's avatar_url at a (versioned) URL.
	SetAvatarURL(ctx context.Context, q db.DBTX, id uuid.UUID, url string) error
	// AudienceOf returns the ids of users who share a private chat or a group
	// with the given user — the set that should learn of their profile edits.
	AudienceOf(ctx context.Context, q db.DBTX, id uuid.UUID) ([]uuid.UUID, error)
	// TouchLastSeen records that the user was last active now (called when
	// their final WebSocket connection closes, to power "last seen" labels).
	TouchLastSeen(ctx context.Context, q db.DBTX, id uuid.UUID) error
	UpdatePasswordHash(ctx context.Context, q db.DBTX, id uuid.UUID, hash string) error
	// AdminResetPasswordHash sets a new hash and forces a change on next
	// login (used when the CEO resets someone's credentials).
	AdminResetPasswordHash(ctx context.Context, q db.DBTX, id uuid.UUID, hash string) error
	UpdateRole(ctx context.Context, q db.DBTX, id uuid.UUID, roleID int) error
	SetActive(ctx context.Context, q db.DBTX, id uuid.UUID, active bool) error
	List(ctx context.Context, q db.DBTX, limit, offset int) ([]User, error)
	// Search returns the active users an actor is allowed to find, excluding
	// themselves. What that means depends on whether the actor has a level at
	// all — see the implementation.
	Search(ctx context.Context, q db.DBTX, actorID uuid.UUID, actorLevel int, query string, limit int) ([]User, error)
	// RegisterLoginFailure atomically increments the failure counter and,
	// when maxAttempts is reached, sets locked_until to lockUntil.
	// Returns the resulting lock timestamp (nil if not locked).
	RegisterLoginFailure(ctx context.Context, q db.DBTX, id uuid.UUID, maxAttempts int, lockUntil time.Time) (*time.Time, error)
	ResetLoginFailures(ctx context.Context, q db.DBTX, id uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

// A basic account has no level, stored as NULL. It is read as access.NoLevel
// (0) so the whole codebase keeps a plain int — the rules in internal/access
// are what give zero its meaning.
const userColumns = `
	id, username::text, display_name, password_hash, COALESCE(role_id, 0), account_kind,
	avatar_url, status, last_seen_at, is_active, failed_login_attempts, locked_until,
	must_change_password, display_name_needs_change, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.RoleID, &u.AccountKind,
		&u.AvatarURL, &u.Status, &u.LastSeenAt, &u.IsActive, &u.FailedLoginAttempts,
		&u.LockedUntil, &u.MustChangePassword, &u.DisplayNameNeedsChange, &u.CreatedAt, &u.UpdatedAt,
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
	// The kind and the level describe one fact, so the level decides and the
	// kind follows. Asking callers to set both invites the two to disagree —
	// and they cannot: the database has a constraint that refuses the
	// combination, which is what caught this in the first place.
	if u.AccountKind == "" {
		u.AccountKind = KindInvited
		if !access.HasLevel(u.RoleID) {
			u.AccountKind = KindBasic
		}
	}

	err := q.QueryRow(ctx, `
		INSERT INTO users (username, display_name, password_hash, role_id, account_kind, must_change_password, display_name_needs_change)
		VALUES ($1, $2, $3, NULLIF($4, 0), $5, $6, $7)
		RETURNING id, status, is_active, failed_login_attempts, must_change_password, created_at, updated_at`,
		u.Username, u.DisplayName, u.PasswordHash, u.RoleID, u.AccountKind, u.MustChangePassword, u.DisplayNameNeedsChange,
	).Scan(&u.ID, &u.Status, &u.IsActive, &u.FailedLoginAttempts, &u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt)

	if err := uniqueViolation(err); err != nil {
		return err
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

// UpdateDisplayName sets a name that has already passed NormalizeDisplayName.
// Choosing a name is also what resolves the migration-46 flag, so both happen
// in one statement — and the unique index sees the new name as unflagged, which
// is what makes a collision fail right here as ErrDisplayNameTaken.
func (r *PostgresRepository) UpdateDisplayName(ctx context.Context, q db.DBTX, id uuid.UUID, displayName string) error {
	tag, err := q.Exec(ctx,
		`UPDATE users SET display_name = $2, display_name_needs_change = false WHERE id = $1`, id, displayName)
	if uerr := uniqueViolation(err); uerr != nil {
		return uerr
	}
	if err != nil {
		return fmt.Errorf("users: update display name: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetAvatarURL(ctx context.Context, q db.DBTX, id uuid.UUID, url string) error {
	tag, err := q.Exec(ctx, `UPDATE users SET avatar_url = $2 WHERE id = $1`, id, url)
	if err != nil {
		return fmt.Errorf("users: set avatar url: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) AudienceOf(ctx context.Context, q db.DBTX, id uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, `
		SELECT CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
		FROM private_chats WHERE user_a_id = $1 OR user_b_id = $1
		UNION
		SELECT gm.user_id FROM group_members gm
		WHERE gm.group_id IN (SELECT group_id FROM group_members WHERE user_id = $1)
		  AND gm.user_id <> $1`,
		id)
	if err != nil {
		return nil, fmt.Errorf("users: audience: %w", err)
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("users: scan audience: %w", err)
		}
		out = append(out, uid)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) TouchLastSeen(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	if _, err := q.Exec(ctx, `UPDATE users SET last_seen_at = now() WHERE id = $1`, id); err != nil {
		return fmt.Errorf("users: touch last seen: %w", err)
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
			&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.RoleID, &u.AccountKind,
			&u.AvatarURL, &u.Status, &u.LastSeenAt, &u.IsActive, &u.FailedLoginAttempts,
			&u.LockedUntil, &u.MustChangePassword, &u.DisplayNameNeedsChange, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("users: scan list row: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Search(ctx context.Context, q db.DBTX, actorID uuid.UUID, actorLevel int, query string, limit int) ([]User, error) {
	// An account with no level has no directory.
	//
	// The clearance rule cannot decide what it may see — it stands outside the
	// hierarchy — and the honest default is not "everyone". A closed corporate
	// messenger must not hand the full staff list to an account that anyone
	// can create by filling in a form, and a prefix search is the same list in
	// slow motion: two letters at a time. So a basic account can only look up
	// someone whose name it already knows, matched whole. Everyone it does
	// find, it may write to (access.CanInitiateChat).
	if !access.HasLevel(actorLevel) {
		return r.searchByFullName(ctx, q, actorID, query, limit)
	}

	// Invited accounts browse the directory as before: everyone at or below
	// their own clearance, matched by username prefix, blank query lists all.
	// Level-less accounts appear here like anyone else — nothing about them is
	// protected by a threshold they do not have.
	rows, err := q.Query(ctx, `SELECT`+userColumns+`
		FROM users
		WHERE is_active = true AND (role_id IS NULL OR role_id >= $1) AND id <> $2
		  AND ($3 = '' OR username LIKE $3 || '%')
		ORDER BY username ASC
		LIMIT $4`, actorLevel, actorID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("users: search: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.RoleID, &u.AccountKind,
			&u.AvatarURL, &u.Status, &u.LastSeenAt, &u.IsActive, &u.FailedLoginAttempts,
			&u.LockedUntil, &u.MustChangePassword, &u.DisplayNameNeedsChange, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("users: scan search row: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// searchByFullName looks someone up by a name the searcher already knows: the
// whole username or the whole display name, case-insensitively and ignoring
// surrounding spaces. Never a prefix, and never a blank query — neither would
// be a lookup, both would be a staff list.
func (r *PostgresRepository) searchByFullName(
	ctx context.Context, q db.DBTX, actorID uuid.UUID, query string, limit int,
) ([]User, error) {
	needle := strings.TrimSpace(query)
	if needle == "" {
		return nil, nil
	}
	rows, err := q.Query(ctx, `SELECT`+userColumns+`
		FROM users
		WHERE is_active = true AND id <> $1
		  AND (username = $2 OR display_name_key = kisy_display_name_key($2))
		ORDER BY username ASC
		LIMIT $3`, actorID, needle, limit)
	if err != nil {
		return nil, fmt.Errorf("users: search by name: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.RoleID, &u.AccountKind,
			&u.AvatarURL, &u.Status, &u.LastSeenAt, &u.IsActive, &u.FailedLoginAttempts,
			&u.LockedUntil, &u.MustChangePassword, &u.DisplayNameNeedsChange, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("users: scan name lookup row: %w", err)
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
