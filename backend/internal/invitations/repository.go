package invitations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// Repository is the persistence port for invitation tokens.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, inv *Invitation) error
	// GetByHashForUpdate locks the row (SELECT ... FOR UPDATE) so that two
	// concurrent registrations cannot both redeem the same token.
	GetByHashForUpdate(ctx context.Context, q db.DBTX, tokenHash string) (*Invitation, error)
	MarkUsed(ctx context.Context, q db.DBTX, id, usedBy uuid.UUID, at time.Time) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, inv *Invitation) error {
	err := q.QueryRow(ctx, `
		INSERT INTO invitation_tokens (token_hash, created_by, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		inv.TokenHash, inv.CreatedBy, inv.CreatedAt, inv.ExpiresAt,
	).Scan(&inv.ID)
	if err != nil {
		return fmt.Errorf("invitations: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByHashForUpdate(ctx context.Context, q db.DBTX, tokenHash string) (*Invitation, error) {
	var inv Invitation
	err := q.QueryRow(ctx, `
		SELECT id, token_hash, created_by, created_at, expires_at, used_at, used_by
		FROM invitation_tokens
		WHERE token_hash = $1
		FOR UPDATE`, tokenHash,
	).Scan(&inv.ID, &inv.TokenHash, &inv.CreatedBy, &inv.CreatedAt, &inv.ExpiresAt, &inv.UsedAt, &inv.UsedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("invitations: get by hash: %w", err)
	}
	return &inv, nil
}

func (r *PostgresRepository) MarkUsed(ctx context.Context, q db.DBTX, id, usedBy uuid.UUID, at time.Time) error {
	tag, err := q.Exec(ctx, `
		UPDATE invitation_tokens SET used_at = $2, used_by = $3
		WHERE id = $1 AND used_at IS NULL`,
		id, at, usedBy,
	)
	if err != nil {
		return fmt.Errorf("invitations: mark used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
