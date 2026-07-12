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
	// ListThreadPage pages one thread's replies (stage K), same semantics.
	ListThreadPage(ctx context.Context, q db.DBTX, rootID uuid.UUID, cur *pagination.Cursor, limit int) ([]Message, error)
	SoftDelete(ctx context.Context, q db.DBTX, id uuid.UUID, at time.Time) error
	// Update replaces a message's text and stamps edited_at, but only for the
	// sender and only while the message is not deleted. Returns ErrNotFound
	// (or ErrForbidden semantics) as zero rows if the guard fails.
	Update(ctx context.Context, q db.DBTX, id, senderID uuid.UUID, text string, at time.Time) (*Message, error)
	// SetPinned pins or unpins a message and returns the fresh row.
	SetPinned(ctx context.Context, q db.DBTX, id uuid.UUID, by *uuid.UUID, at *time.Time) (*Message, error)
	// SetExpiry sets or clears a message's self-destruct time, but only for
	// the sender and only while the message is not deleted.
	SetExpiry(ctx context.Context, q db.DBTX, id, senderID uuid.UUID, at *time.Time) (*Message, error)
	// ListPinned returns the chat's pinned messages, most recently pinned first.
	ListPinned(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID) ([]Message, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const messageColumns = `id, chat_type, chat_id, sender_id, text, reply_to, is_deleted, deleted_at, edited_at, pinned_at, pinned_by, created_at, ciphertext, alg, epoch, content_kind, forwarded_from_sender_id, forwarded_from_sender_name, scheduled_message_id, expires_at, thread_root_id, thread_reply_count, thread_last_reply_at`

func scanMessage(row pgx.Row) (*Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.ChatType, &m.ChatID, &m.SenderID, &m.Text, &m.ReplyTo,
		&m.IsDeleted, &m.DeletedAt, &m.EditedAt, &m.PinnedAt, &m.PinnedBy, &m.CreatedAt,
		&m.Ciphertext, &m.Alg, &m.Epoch, &m.ContentKind,
		&m.ForwardedFromSenderID, &m.ForwardedFromSenderName, &m.ScheduledMessageID, &m.ExpiresAt,
		&m.ThreadRootID, &m.ThreadReplyCount, &m.ThreadLastReplyAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messages: scan: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, m *Message) error {
	// One statement: the insert and the root's denormalized reply counters
	// (stage K) land atomically even when q is the plain pool. The bump CTE
	// matches nothing for non-thread messages.
	err := q.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO messages (chat_type, chat_id, sender_id, text, reply_to, ciphertext, alg, epoch, content_kind,
			                      forwarded_from_message_id, forwarded_from_sender_id, forwarded_from_sender_name,
			                      scheduled_message_id, expires_at, thread_root_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			RETURNING id, is_deleted, created_at, thread_root_id
		), bump AS (
			UPDATE messages r
			SET thread_reply_count = r.thread_reply_count + 1,
			    thread_last_reply_at = ins.created_at
			FROM ins
			WHERE r.id = ins.thread_root_id
		)
		SELECT id, is_deleted, created_at FROM ins`,
		m.ChatType, m.ChatID, m.SenderID, m.Text, m.ReplyTo, m.Ciphertext, m.Alg, m.Epoch, m.ContentKind,
		m.ForwardedFromMessageID, m.ForwardedFromSenderID, m.ForwardedFromSenderName,
		m.ScheduledMessageID, m.ExpiresAt, m.ThreadRootID,
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
	// Thread replies (stage K) live only in their thread page, never in the
	// main feed.
	if cur == nil {
		rows, err = q.Query(ctx, `
			SELECT `+messageColumns+` FROM messages
			WHERE chat_type = $1 AND chat_id = $2 AND thread_root_id IS NULL
			ORDER BY created_at DESC, id DESC
			LIMIT $3`, chatType, chatID, limit+1)
	} else {
		rows, err = q.Query(ctx, `
			SELECT `+messageColumns+` FROM messages
			WHERE chat_type = $1 AND chat_id = $2 AND thread_root_id IS NULL
			  AND (created_at, id) < ($3, $4)
			ORDER BY created_at DESC, id DESC
			LIMIT $5`, chatType, chatID, cur.CreatedAt, cur.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("messages: list page: %w", err)
	}
	return collectMessages(rows)
}

// ListThreadPage returns up to limit+1 replies of one thread, newest first
// (same cursor semantics as ListPage — the client renders oldest-first).
func (r *PostgresRepository) ListThreadPage(ctx context.Context, q db.DBTX, rootID uuid.UUID, cur *pagination.Cursor, limit int) ([]Message, error) {
	var rows pgx.Rows
	var err error
	if cur == nil {
		rows, err = q.Query(ctx, `
			SELECT `+messageColumns+` FROM messages
			WHERE thread_root_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`, rootID, limit+1)
	} else {
		rows, err = q.Query(ctx, `
			SELECT `+messageColumns+` FROM messages
			WHERE thread_root_id = $1
			  AND (created_at, id) < ($2, $3)
			ORDER BY created_at DESC, id DESC
			LIMIT $4`, rootID, cur.CreatedAt, cur.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("messages: list thread: %w", err)
	}
	return collectMessages(rows)
}

// collectMessages drains a query over messageColumns into a slice.
func collectMessages(rows pgx.Rows) ([]Message, error) {
	defer rows.Close()
	var out []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, q db.DBTX, id, senderID uuid.UUID, text string, at time.Time) (*Message, error) {
	row := q.QueryRow(ctx, `
		UPDATE messages SET text = $3, edited_at = $4
		WHERE id = $1 AND sender_id = $2 AND is_deleted = false
		RETURNING `+messageColumns,
		id, senderID, text, at)
	m, err := scanMessage(row)
	if errors.Is(err, ErrNotFound) {
		// No row matched the id+sender+not-deleted guard.
		return nil, ErrForbidden
	}
	return m, err
}

func (r *PostgresRepository) SetPinned(ctx context.Context, q db.DBTX, id uuid.UUID, by *uuid.UUID, at *time.Time) (*Message, error) {
	row := q.QueryRow(ctx, `
		UPDATE messages SET pinned_at = $2, pinned_by = $3
		WHERE id = $1 AND is_deleted = false
		RETURNING `+messageColumns, id, at, by)
	return scanMessage(row)
}

func (r *PostgresRepository) SetExpiry(ctx context.Context, q db.DBTX, id, senderID uuid.UUID, at *time.Time) (*Message, error) {
	row := q.QueryRow(ctx, `
		UPDATE messages SET expires_at = $3
		WHERE id = $1 AND sender_id = $2 AND is_deleted = false
		RETURNING `+messageColumns, id, senderID, at)
	m, err := scanMessage(row)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrForbidden
	}
	return m, err
}

func (r *PostgresRepository) ListPinned(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID) ([]Message, error) {
	rows, err := q.Query(ctx, `
		SELECT `+messageColumns+` FROM messages
		WHERE chat_type = $1 AND chat_id = $2 AND pinned_at IS NOT NULL AND is_deleted = false
		ORDER BY pinned_at DESC`, chatType, chatID)
	if err != nil {
		return nil, fmt.Errorf("messages: list pinned: %w", err)
	}
	return collectMessages(rows)
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
