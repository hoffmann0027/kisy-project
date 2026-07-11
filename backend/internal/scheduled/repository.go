package scheduled

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const columns = `id, chat_type, chat_id, sender_id, text, ciphertext, alg, epoch, content_kind, reply_to, attachment_ids, send_at, status, sent_message_id, created_at`

func scan(row pgx.Row) (*Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.ChatType, &m.ChatID, &m.SenderID, &m.Text, &m.Ciphertext,
		&m.Alg, &m.Epoch, &m.ContentKind, &m.ReplyTo, &m.AttachmentIDs,
		&m.SendAt, &m.Status, &m.SentMessageID, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scheduled: scan: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, m *Message) error {
	if m.AttachmentIDs == nil {
		m.AttachmentIDs = []uuid.UUID{}
	}
	err := q.QueryRow(ctx, `
		INSERT INTO scheduled_messages
			(chat_type, chat_id, sender_id, text, ciphertext, alg, epoch, content_kind, reply_to, attachment_ids, send_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, status, created_at`,
		m.ChatType, m.ChatID, m.SenderID, m.Text, m.Ciphertext, m.Alg, m.Epoch, m.ContentKind,
		m.ReplyTo, m.AttachmentIDs, m.SendAt,
	).Scan(&m.ID, &m.Status, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("scheduled: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetPending(ctx context.Context, q db.DBTX, id, senderID uuid.UUID) (*Message, error) {
	return scan(q.QueryRow(ctx, `
		SELECT `+columns+` FROM scheduled_messages
		WHERE id = $1 AND sender_id = $2 AND status = 'pending'`, id, senderID))
}

func (r *PostgresRepository) ListForSender(ctx context.Context, q db.DBTX, senderID uuid.UUID) ([]Message, error) {
	// Pending first (soonest on top), then the recent history.
	rows, err := q.Query(ctx, `
		SELECT `+columns+` FROM scheduled_messages
		WHERE sender_id = $1
		ORDER BY (status <> 'pending'), send_at ASC
		LIMIT 200`, senderID)
	if err != nil {
		return nil, fmt.Errorf("scheduled: list: %w", err)
	}
	defer rows.Close()
	return collect(rows)
}

func collect(rows pgx.Rows) ([]Message, error) {
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChatType, &m.ChatID, &m.SenderID, &m.Text, &m.Ciphertext,
			&m.Alg, &m.Epoch, &m.ContentKind, &m.ReplyTo, &m.AttachmentIDs,
			&m.SendAt, &m.Status, &m.SentMessageID, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scheduled: scan row: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CountPending(ctx context.Context, q db.DBTX, senderID uuid.UUID) (int, error) {
	var n int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*) FROM scheduled_messages WHERE sender_id = $1 AND status = 'pending'`, senderID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("scheduled: count: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) Update(ctx context.Context, q db.DBTX, m *Message) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE scheduled_messages
		SET text = $3, ciphertext = $4, alg = $5, epoch = $6, content_kind = $7, send_at = $8, updated_at = now()
		WHERE id = $1 AND sender_id = $2 AND status = 'pending'`,
		m.ID, m.SenderID, m.Text, m.Ciphertext, m.Alg, m.Epoch, m.ContentKind, m.SendAt)
	if err != nil {
		return false, fmt.Errorf("scheduled: update: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) DeletePending(ctx context.Context, q db.DBTX, id, senderID uuid.UUID) (bool, error) {
	tag, err := q.Exec(ctx, `
		DELETE FROM scheduled_messages WHERE id = $1 AND sender_id = $2 AND status = 'pending'`, id, senderID)
	if err != nil {
		return false, fmt.Errorf("scheduled: delete: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) ClaimDue(ctx context.Context, q db.DBTX, now time.Time, limit int) ([]Message, error) {
	// SKIP LOCKED: concurrent workers (several backend replicas) each claim a
	// disjoint set; a claimed row stays invisible to others until commit.
	rows, err := q.Query(ctx, `
		SELECT `+columns+` FROM scheduled_messages
		WHERE status = 'pending' AND send_at <= $1
		ORDER BY send_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("scheduled: claim due: %w", err)
	}
	defer rows.Close()
	return collect(rows)
}

func (r *PostgresRepository) MarkSent(ctx context.Context, q db.DBTX, id, messageID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE scheduled_messages SET status = 'sent', sent_message_id = $2, updated_at = now() WHERE id = $1`,
		id, messageID)
	if err != nil {
		return fmt.Errorf("scheduled: mark sent: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkCanceled(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	// Canceling drops the frozen content: only the fact of cancellation stays.
	_, err := q.Exec(ctx, `
		UPDATE scheduled_messages
		SET status = 'canceled', text = NULL, ciphertext = NULL, attachment_ids = '{}', updated_at = now()
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("scheduled: mark canceled: %w", err)
	}
	return nil
}

// StartWorker runs ProcessDue on a fixed interval until ctx is cancelled
// (same lifecycle pattern as the attachments session reaper).
func (s *Service) StartWorker(ctx context.Context, interval time.Duration, log *slog.Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Drain everything due, batch by batch, so a backlog after
				// downtime clears in one tick.
				for {
					n, err := s.ProcessDue(ctx, time.Now(), 50)
					if err != nil {
						if ctx.Err() == nil {
							log.Warn("scheduled: worker pass failed", "error", err)
						}
						break
					}
					if n > 0 {
						log.Info("scheduled: processed due messages", "count", n)
					}
					if n < 50 {
						break
					}
				}
			}
		}
	}()
}
