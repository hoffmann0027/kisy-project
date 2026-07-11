package chatmedia

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/attachments"
	"kisy-backend/internal/platform/db"
	"kisy-backend/pkg/pagination"
)

// Repository reads message-derived content of one chat, newest first, with
// (created_at, id) cursor pagination. Queries always fetch limit+1 rows so
// the service can detect further pages.
type Repository interface {
	ListAttachments(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, kinds []string, cur *pagination.Cursor, limit int) ([]Item, error)
	// ListLinkMessages returns plaintext messages that contain a URL; link
	// extraction happens in the service (SQL only pre-filters).
	ListLinkMessages(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, cur *pagination.Cursor, limit int) ([]linkSource, error)
}

type linkSource struct {
	MessageID uuid.UUID
	SenderID  uuid.UUID
	Text      string
	CreatedAt pagination.Cursor // reuse the (CreatedAt, ID) pair shape
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) ListAttachments(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, kinds []string, cur *pagination.Cursor, limit int) ([]Item, error) {
	// The cursor points at an attachment row (a.created_at, a.id).
	var rows pgx.Rows
	var err error
	const base = `
		SELECT a.id, a.file_name, a.mime_type, a.size_bytes, a.kind, a.duration_ms, a.waveform, a.width, a.height,
		       m.id, m.sender_id, a.created_at
		FROM attachments a
		JOIN messages m ON m.id = a.message_id
		WHERE m.chat_type = $1 AND m.chat_id = $2 AND m.is_deleted = false
		  AND a.kind = ANY($3)`
	if cur == nil {
		rows, err = q.Query(ctx, base+`
			ORDER BY a.created_at DESC, a.id DESC
			LIMIT $4`, chatType, chatID, kinds, limit+1)
	} else {
		rows, err = q.Query(ctx, base+`
			  AND (a.created_at, a.id) < ($4, $5)
			ORDER BY a.created_at DESC, a.id DESC
			LIMIT $6`, chatType, chatID, kinds, cur.CreatedAt, cur.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("chatmedia: list attachments: %w", err)
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var it Item
		var a attachments.DTO
		if err := rows.Scan(&a.ID, &a.FileName, &a.MimeType, &a.SizeBytes, &a.Kind,
			&a.DurationMs, &a.Waveform, &a.Width, &a.Height,
			&it.MessageID, &it.SenderID, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("chatmedia: scan attachment: %w", err)
		}
		a.IsImage = a.Kind == attachments.KindImage
		a.URL = "/api/v1/attachments/" + a.ID.String()
		it.Attachment = a
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListLinkMessages(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, cur *pagination.Cursor, limit int) ([]linkSource, error) {
	// Cheap SQL pre-filter; precise URL extraction happens in Go. E2EE
	// messages have NULL text and never match — the server cannot (and must
	// not) look inside ciphertext.
	var rows pgx.Rows
	var err error
	const base = `
		SELECT id, sender_id, text, created_at
		FROM messages
		WHERE chat_type = $1 AND chat_id = $2 AND is_deleted = false
		  AND text ~* 'https?://'`
	if cur == nil {
		rows, err = q.Query(ctx, base+`
			ORDER BY created_at DESC, id DESC
			LIMIT $3`, chatType, chatID, limit+1)
	} else {
		rows, err = q.Query(ctx, base+`
			  AND (created_at, id) < ($3, $4)
			ORDER BY created_at DESC, id DESC
			LIMIT $5`, chatType, chatID, cur.CreatedAt, cur.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("chatmedia: list link messages: %w", err)
	}
	defer rows.Close()

	var out []linkSource
	for rows.Next() {
		var s linkSource
		if err := rows.Scan(&s.MessageID, &s.SenderID, &s.Text, &s.CreatedAt.CreatedAt); err != nil {
			return nil, fmt.Errorf("chatmedia: scan link message: %w", err)
		}
		s.CreatedAt.ID = s.MessageID
		out = append(out, s)
	}
	return out, rows.Err()
}
