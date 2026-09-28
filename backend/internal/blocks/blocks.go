// Package blocks lets one account shut another out of its life here.
//
// Why it exists (audit E-02): registration is open, the feed is public and
// anyone may write to anyone they can find. Rate limits and the new-account
// quarantine make spam expensive; blocking is what a person does about the one
// individual who is after them specifically, without waiting for a moderator.
//
// Two rules shape everything below:
//
//   - A block is one-sided and silent. The blocked person is not told and sees
//     no difference in wording: refusals read the same as any other "you
//     cannot write here". Telling them would turn blocking into a provocation.
//   - It works in both directions for visibility. If either side has blocked
//     the other, neither sees the other in the feed, in search or in calls —
//     the blocker should not have to keep seeing the person they blocked.
package blocks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/db"
)

var (
	// ErrSelf: blocking yourself is meaningless.
	ErrSelf = errors.New("blocks: cannot block yourself")
	// ErrBlocked is what a refused action returns. The caller turns it into a
	// message that never says which side did the blocking.
	ErrBlocked = errors.New("blocks: blocked")
)

// Block is one entry of the list a person keeps.
type Block struct {
	UserID    uuid.UUID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Service struct {
	pool  *pgxpool.Pool
	audit audit.Recorder
}

func NewService(pool *pgxpool.Pool, rec audit.Recorder) *Service {
	return &Service{pool: pool, audit: rec}
}

// ActorMeta is who is blocking, and from where.
type ActorMeta struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Add blocks target for actor. Blocking twice is not an error: the person
// pressed the button, and the outcome they asked for holds.
func (s *Service) Add(ctx context.Context, actor ActorMeta, target uuid.UUID) error {
	if actor.UserID == target {
		return ErrSelf
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("blocks: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, actor.UserID, target); err != nil {
		return fmt.Errorf("blocks: add: %w", err)
	}
	if err := s.record(ctx, tx, actor, target, "user.block"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Remove lifts the block.
func (s *Service) Remove(ctx context.Context, actor ActorMeta, target uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("blocks: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2`, actor.UserID, target); err != nil {
		return fmt.Errorf("blocks: remove: %w", err)
	}
	if err := s.record(ctx, tx, actor, target, "user.unblock"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) record(ctx context.Context, q db.DBTX, actor ActorMeta, target uuid.UUID, action string) error {
	if s.audit == nil {
		return nil
	}
	targetID := target
	var session *uuid.UUID
	if actor.SessionID != uuid.Nil {
		session = &actor.SessionID
	}
	return s.audit.Record(ctx, q, audit.Event{
		ActorID:    &actor.UserID,
		Action:     action,
		TargetType: "user",
		TargetID:   &targetID,
		IPHash:     actor.IPHash,
		SessionID:  session,
		RequestID:  actor.RequestID,
	})
}

// List returns who this account has blocked, newest first.
func (s *Service) List(ctx context.Context, blocker uuid.UUID) ([]Block, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT blocked_id, created_at FROM user_blocks
		WHERE blocker_id = $1 ORDER BY created_at DESC`, blocker)
	if err != nil {
		return nil, fmt.Errorf("blocks: list: %w", err)
	}
	defer rows.Close()

	out := []Block{}
	for rows.Next() {
		var b Block
		if err := rows.Scan(&b.UserID, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("blocks: scan: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Between reports whether either side has blocked the other. This is the check
// on the path of a message, so it is one round trip and nothing more.
func (s *Service) Between(ctx context.Context, a, b uuid.UUID) (bool, error) {
	var blocked bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)`, a, b).Scan(&blocked)
	if err != nil {
		return false, fmt.Errorf("blocks: between: %w", err)
	}
	return blocked, nil
}

// Hidden lists everyone the viewer should not see and who should not see the
// viewer: both directions, in one list. Used to filter the feed, the walls and
// people search, where the alternative is a per-row check.
func (s *Service) Hidden(ctx context.Context, viewer uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT blocked_id FROM user_blocks WHERE blocker_id = $1
		UNION
		SELECT blocker_id FROM user_blocks WHERE blocked_id = $1`, viewer)
	if err != nil {
		return nil, fmt.Errorf("blocks: hidden: %w", err)
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("blocks: scan hidden: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
