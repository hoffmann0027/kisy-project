package chats

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// Repository is the persistence port for private chats.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, c *PrivateChat) error
	GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*PrivateChat, error)
	FindByPair(ctx context.Context, q db.DBTX, userA, userB uuid.UUID) (*PrivateChat, error)
	ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]PrivateChat, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const chatColumns = `id, user_a_id, user_b_id, initiated_by, created_at`

func scanChat(row pgx.Row) (*PrivateChat, error) {
	var c PrivateChat
	err := row.Scan(&c.ID, &c.UserAID, &c.UserBID, &c.InitiatedBy, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("chats: scan: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, c *PrivateChat) error {
	err := q.QueryRow(ctx, `
		INSERT INTO private_chats (user_a_id, user_b_id, initiated_by)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		c.UserAID, c.UserBID, c.InitiatedBy,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("chats: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*PrivateChat, error) {
	return scanChat(q.QueryRow(ctx, `SELECT `+chatColumns+` FROM private_chats WHERE id = $1`, id))
}

// FindByPair looks up the (unordered) conversation between two users.
func (r *PostgresRepository) FindByPair(ctx context.Context, q db.DBTX, userA, userB uuid.UUID) (*PrivateChat, error) {
	return scanChat(q.QueryRow(ctx, `
		SELECT `+chatColumns+` FROM private_chats
		WHERE (user_a_id = $1 AND user_b_id = $2) OR (user_a_id = $2 AND user_b_id = $1)`,
		userA, userB))
}

func (r *PostgresRepository) ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]PrivateChat, error) {
	rows, err := q.Query(ctx, `
		SELECT `+chatColumns+` FROM private_chats
		WHERE user_a_id = $1 OR user_b_id = $1
		ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("chats: list for user: %w", err)
	}
	defer rows.Close()

	var out []PrivateChat
	for rows.Next() {
		var c PrivateChat
		if err := rows.Scan(&c.ID, &c.UserAID, &c.UserBID, &c.InitiatedBy, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("chats: scan row: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
