package messages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
	"kisy-backend/pkg/pagination"
)

// Repository is the persistence port for messages.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, m *Message) error
	GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Message, error)
	// ListPage returns up to limit+1 messages of a chat, newest first,
	// starting after the optional cursor. The extra row lets the caller
	// detect whether more pages exist.
	ListPage(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, cur *pagination.Cursor, limit int) ([]Message, error)
	SoftDelete(ctx context.Context, q db.DBTX, id uuid.UUID, at time.Time) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const messageColumns = `id, chat_type, chat_id, sender_id, text, reply_to, is_deleted, deleted_at, created_at`

func scanMessage(row pgx.Row) (*Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.ChatType, &m.ChatID, &m.SenderID, &m.Text, &m.ReplyTo,
		&m.IsDeleted, &m.DeletedAt, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messages: scan: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, m *Message) error {
	err := q.QueryRow(ctx, `
		INSERT INTO messages (chat_type, chat_id, sender_id, text, reply_to)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, is_deleted, created_at`,
		m.ChatType, m.ChatID, m.SenderID, m.Text, m.ReplyTo,
	).Scan(&m.ID, &m.IsDeleted, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("messages: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Message, error) {
	return scanMessage(q.QueryRow(ctx, `SELECT `+messageColumns+` FROM messages WHERE id = $1`, id))
}

func (r *PostgresRepository) ListPage(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, cur *pagination.Cursor, limit int) ([]Message, error) {
	// Stable ordering by (created_at, id) descending; the row-value
	// comparison walks strictly past the cursor position.
	var rows pgx.Rows
	var err error
	if cur == nil {
		rows, err = q.Query(ctx, `
			SELECT `+messageColumns+` FROM messages
			WHERE chat_type = $1 AND chat_id = $2
			ORDER BY created_at DESC, id DESC
			LIMIT $3`, chatType, chatID, limit+1)
	} else {
		rows, err = q.Query(ctx, `
			SELECT `+messageColumns+` FROM messages
			WHERE chat_type = $1 AND chat_id = $2
			  AND (created_at, id) < ($3, $4)
			ORDER BY created_at DESC, id DESC
			LIMIT $5`, chatType, chatID, cur.CreatedAt, cur.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("messages: list page: %w", err)
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChatType, &m.ChatID, &m.SenderID, &m.Text, &m.ReplyTo,
			&m.IsDeleted, &m.DeletedAt, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("messages: scan row: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, q db.DBTX, id uuid.UUID, at time.Time) error {
	tag, err := q.Exec(ctx, `
		UPDATE messages SET is_deleted = true, deleted_at = $2, text = NULL
		WHERE id = $1 AND is_deleted = false`, id, at)
	if err != nil {
		return fmt.Errorf("messages: soft delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Either the message does not exist or it was already deleted;
		// callers treat both as not-found for idempotency.
		return ErrNotFound
	}
	return nil
}
