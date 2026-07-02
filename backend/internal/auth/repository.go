package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// SessionRepository is the persistence port for refresh sessions.
type SessionRepository interface {
	Create(ctx context.Context, q db.DBTX, s *Session) error
	GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Session, error)
	Rotate(ctx context.Context, q db.DBTX, id uuid.UUID, newHash string, lastUsed, expires time.Time) error
	Revoke(ctx context.Context, q db.DBTX, id uuid.UUID, at time.Time) error
	RevokeAllForUser(ctx context.Context, q db.DBTX, userID uuid.UUID, at time.Time) (int64, error)
	RevokeAllForUserExcept(ctx context.Context, q db.DBTX, userID, keep uuid.UUID, at time.Time) (int64, error)
}

var ErrSessionNotFound = errors.New("auth: session not found")

type PostgresSessionRepository struct{}

func NewPostgresSessionRepository() *PostgresSessionRepository { return &PostgresSessionRepository{} }

func (r *PostgresSessionRepository) Create(ctx context.Context, q db.DBTX, s *Session) error {
	err := q.QueryRow(ctx, `
		INSERT INTO sessions (user_id, refresh_token_hash, device_name, user_agent, ip_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, last_used_at`,
		s.UserID, s.RefreshTokenHash, s.DeviceName, s.UserAgent, s.IPHash, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt, &s.LastUsedAt)
	if err != nil {
		return fmt.Errorf("auth: create session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Session, error) {
	var s Session
	err := q.QueryRow(ctx, `
		SELECT id, user_id, refresh_token_hash, device_name, user_agent, ip_hash,
		       created_at, last_used_at, expires_at, revoked_at
		FROM sessions WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.DeviceName, &s.UserAgent, &s.IPHash,
		&s.CreatedAt, &s.LastUsedAt, &s.ExpiresAt, &s.RevokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get session: %w", err)
	}
	return &s, nil
}

func (r *PostgresSessionRepository) Rotate(ctx context.Context, q db.DBTX, id uuid.UUID, newHash string, lastUsed, expires time.Time) error {
	tag, err := q.Exec(ctx, `
		UPDATE sessions SET refresh_token_hash = $2, last_used_at = $3, expires_at = $4
		WHERE id = $1 AND revoked_at IS NULL`,
		id, newHash, lastUsed, expires,
	)
	if err != nil {
		return fmt.Errorf("auth: rotate session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *PostgresSessionRepository) Revoke(ctx context.Context, q db.DBTX, id uuid.UUID, at time.Time) error {
	tag, err := q.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`, id, at)
	if err != nil {
		return fmt.Errorf("auth: revoke session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *PostgresSessionRepository) RevokeAllForUser(ctx context.Context, q db.DBTX, userID uuid.UUID, at time.Time) (int64, error) {
	tag, err := q.Exec(ctx, `
		UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, at)
	if err != nil {
		return 0, fmt.Errorf("auth: revoke all sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *PostgresSessionRepository) RevokeAllForUserExcept(ctx context.Context, q db.DBTX, userID, keep uuid.UUID, at time.Time) (int64, error) {
	tag, err := q.Exec(ctx, `
		UPDATE sessions SET revoked_at = $3 WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL`,
		userID, keep, at)
	if err != nil {
		return 0, fmt.Errorf("auth: revoke other sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
