// Package quota bounds how much a single account — and a single community —
// can put into storage, and how fast an account can publish posts.
//
// Why it exists (audit A-07): registration is open, uploads had only a per-file
// ceiling, and without an object store the bytes live in Postgres. One free
// account could fill the database — and a full database stops the whole
// messenger, not just uploads. Per-file limits bound one request; these bound
// the account.
//
// The rules are written once here and called from every path that stores
// bytes (attachments, chunked uploads, forwarded copies, notes, post media) and
// from post creation. Each check runs inside the caller's transaction under a
// per-account advisory lock, so parallel uploads cannot each see the same
// "still under quota" and together overshoot it.
package quota

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/access"
	"kisy-backend/internal/platform/db"
)

var (
	// ErrUserStorage: the account's stored bytes would exceed its quota.
	ErrUserStorage = errors.New("quota: account storage quota exceeded")
	// ErrCommunityStorage: the community's post media would exceed its quota.
	ErrCommunityStorage = errors.New("quota: community storage quota exceeded")
	// ErrPostRate: the account has published its allowance of posts this hour.
	ErrPostRate = errors.New("quota: too many posts this hour")
)

// Policy is the operator-set limits (config: QUOTA_*, POSTS_PER_HOUR_*).
// Zero means "no limit". The CEO is never limited: the account that runs the
// deployment must be able to clean it up.
type Policy struct {
	// UserBytesBasic / UserBytesInvited cap everything one account has stored:
	// message attachments (forwarded copies included — without an object store
	// a forward duplicates the bytes), pending chunked uploads, note files and
	// post media.
	UserBytesBasic   int64
	UserBytesInvited int64
	// CommunityBytes caps the post media stored in one community.
	CommunityBytes int64
	// PostsPerHourBasic / PostsPerHourInvited cap new posts per account over
	// a rolling hour.
	PostsPerHourBasic   int
	PostsPerHourInvited int
}

// Checker applies a Policy. The zero value enforces nothing.
type Checker struct {
	policy Policy
}

func New(p Policy) *Checker { return &Checker{policy: p} }

// lock serialises every quota decision for one key until the surrounding
// transaction ends. Called outside a transaction it still runs (and still
// checks), it just cannot stop a concurrent request from racing it.
func lock(ctx context.Context, q db.DBTX, key string) error {
	if _, err := q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
		return fmt.Errorf("quota: lock: %w", err)
	}
	return nil
}

type account struct {
	ceo     bool
	invited bool
}

// accountOf reads the account's standing from the database, not from a token:
// a quota must follow the account as it is now.
func accountOf(ctx context.Context, q db.DBTX, userID uuid.UUID) (account, error) {
	var level *int
	err := q.QueryRow(ctx, `SELECT role_id FROM users WHERE id = $1`, userID).Scan(&level)
	if errors.Is(err, pgx.ErrNoRows) {
		return account{}, nil // no such user: treated as the most restricted
	}
	if err != nil {
		return account{}, fmt.Errorf("quota: load account: %w", err)
	}
	if level == nil {
		return account{}, nil
	}
	return account{ceo: access.IsCEO(*level), invited: access.HasLevel(*level)}, nil
}

// UserBytes returns how many bytes the account currently holds in storage.
func UserBytes(ctx context.Context, q db.DBTX, userID uuid.UUID) (int64, error) {
	var used int64
	err := q.QueryRow(ctx, `
		SELECT
		  COALESCE((SELECT sum(size_bytes) FROM attachments WHERE uploaded_by = $1), 0)
		+ COALESCE((SELECT sum(declared_bytes) FROM attachment_upload_sessions
		            WHERE uploader = $1 AND expires_at > now()), 0)
		+ COALESCE((SELECT sum(file_size) FROM notes WHERE user_id = $1), 0)
		+ COALESCE((SELECT sum(pm.size_bytes) FROM post_media pm
		            JOIN posts p ON p.id = pm.post_id WHERE p.author_id = $1), 0)`,
		userID).Scan(&used)
	if err != nil {
		return 0, fmt.Errorf("quota: user usage: %w", err)
	}
	return used, nil
}

// CommunityBytes returns how many bytes of post media a community holds.
func CommunityBytes(ctx context.Context, q db.DBTX, communityID uuid.UUID) (int64, error) {
	var used int64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(pm.size_bytes), 0) FROM post_media pm
		JOIN posts p ON p.id = pm.post_id WHERE p.community_id = $1`,
		communityID).Scan(&used)
	if err != nil {
		return 0, fmt.Errorf("quota: community usage: %w", err)
	}
	return used, nil
}

// ReserveUserBytes allows storing add more bytes for the account, or returns
// ErrUserStorage. Run it in the transaction that writes the bytes.
func (c *Checker) ReserveUserBytes(ctx context.Context, q db.DBTX, userID uuid.UUID, add int64) error {
	if c == nil {
		return nil
	}
	acc, err := accountOf(ctx, q, userID)
	if err != nil {
		return err
	}
	limit := c.policy.UserBytesBasic
	if acc.invited {
		limit = c.policy.UserBytesInvited
	}
	if acc.ceo || limit <= 0 {
		return nil
	}
	if err := lock(ctx, q, "quota:user:"+userID.String()); err != nil {
		return err
	}
	used, err := UserBytes(ctx, q, userID)
	if err != nil {
		return err
	}
	if used+add > limit {
		return ErrUserStorage
	}
	return nil
}

// ReserveCommunityBytes allows add more bytes of post media in the community.
func (c *Checker) ReserveCommunityBytes(ctx context.Context, q db.DBTX, communityID uuid.UUID, add int64) error {
	if c == nil || c.policy.CommunityBytes <= 0 {
		return nil
	}
	if err := lock(ctx, q, "quota:community:"+communityID.String()); err != nil {
		return err
	}
	used, err := CommunityBytes(ctx, q, communityID)
	if err != nil {
		return err
	}
	if used+add > c.policy.CommunityBytes {
		return ErrCommunityStorage
	}
	return nil
}

// ReservePost allows the account one more post this hour, or ErrPostRate.
// Counted from the posts table itself (deleted posts included — deleting and
// reposting must not reset the allowance), so it needs no extra store and
// survives restarts and several instances.
func (c *Checker) ReservePost(ctx context.Context, q db.DBTX, userID uuid.UUID) error {
	if c == nil {
		return nil
	}
	acc, err := accountOf(ctx, q, userID)
	if err != nil {
		return err
	}
	limit := c.policy.PostsPerHourBasic
	if acc.invited {
		limit = c.policy.PostsPerHourInvited
	}
	if acc.ceo || limit <= 0 {
		return nil
	}
	if err := lock(ctx, q, "quota:posts:"+userID.String()); err != nil {
		return err
	}
	var recent int
	err = q.QueryRow(ctx, `
		SELECT count(*) FROM posts
		WHERE author_id = $1 AND created_at > now() - interval '1 hour'`, userID).Scan(&recent)
	if err != nil {
		return fmt.Errorf("quota: recent posts: %w", err)
	}
	if recent >= limit {
		return ErrPostRate
	}
	return nil
}

// Describe maps a quota error to the HTTP status and user-facing message every
// handler answers with, so the wording is written once. ok is false for any
// other error.
func Describe(err error) (status int, message string, ok bool) {
	switch {
	case errors.Is(err, ErrUserStorage):
		return http.StatusRequestEntityTooLarge, "Место для ваших файлов закончилось: удалите старые вложения или заметки", true
	case errors.Is(err, ErrCommunityStorage):
		return http.StatusRequestEntityTooLarge, "Место для медиа в этом сообществе закончилось", true
	case errors.Is(err, ErrPostRate):
		return http.StatusTooManyRequests, "Слишком много постов за час — попробуйте позже", true
	}
	return 0, "", false
}

// Is reports whether err is one of this package's limits.
func Is(err error) bool {
	_, _, ok := Describe(err)
	return ok
}
