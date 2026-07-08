package calls

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/db"
)

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, log CallLog) error {
	_, err := q.Exec(ctx, `
		INSERT INTO call_logs (id, caller_id, callee_id, chat_id, status, started_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		log.ID, log.CallerID, log.CalleeID, log.ChatID, log.Status, log.StartedAt)
	if err != nil {
		return fmt.Errorf("calls: create log: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Finalize(ctx context.Context, q db.DBTX, id uuid.UUID, status string, answeredAt, endedAt *time.Time, durationSeconds int) error {
	_, err := q.Exec(ctx, `
		UPDATE call_logs
		SET status = $2, answered_at = $3, ended_at = $4, duration_seconds = $5
		WHERE id = $1`,
		id, status, answeredAt, endedAt, durationSeconds)
	if err != nil {
		return fmt.Errorf("calls: finalize log: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID, limit, offset int) ([]CallLogRow, error) {
	rows, err := q.Query(ctx, `
		SELECT c.id, c.caller_id, c.callee_id, c.chat_id, c.status,
		       c.started_at, c.answered_at, c.ended_at, c.duration_seconds,
		       uc.display_name, uc.avatar_url, ue.display_name, ue.avatar_url
		FROM call_logs c
		JOIN users uc ON uc.id = c.caller_id
		JOIN users ue ON ue.id = c.callee_id
		WHERE c.caller_id = $1 OR c.callee_id = $1
		ORDER BY c.started_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("calls: list logs: %w", err)
	}
	defer rows.Close()

	var out []CallLogRow
	for rows.Next() {
		var row CallLogRow
		if err := rows.Scan(&row.ID, &row.CallerID, &row.CalleeID, &row.ChatID, &row.Status,
			&row.StartedAt, &row.AnsweredAt, &row.EndedAt, &row.DurationSeconds,
			&row.CallerName, &row.CallerAvatar, &row.CalleeName, &row.CalleeAvatar); err != nil {
			return nil, fmt.Errorf("calls: scan log: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
