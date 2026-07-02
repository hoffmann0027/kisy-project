package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LogEntry is a read view of one audit row.
type LogEntry struct {
	ID         uuid.UUID       `json:"id"`
	ActorID    *uuid.UUID      `json:"actorId"`
	Action     string          `json:"action"`
	TargetType *string         `json:"targetType"`
	TargetID   *uuid.UUID      `json:"targetId"`
	RequestID  *string         `json:"requestId"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// Reader queries the append-only audit trail for the admin surface.
type Reader struct {
	pool *pgxpool.Pool
}

func NewReader(pool *pgxpool.Pool) *Reader { return &Reader{pool: pool} }

// List returns audit entries newest-first with offset pagination,
// optionally filtered by action.
func (r *Reader) List(ctx context.Context, action string, limit, offset int) ([]LogEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_id, action, target_type, target_id, request_id, metadata, created_at
		FROM audit_logs
		WHERE ($1 = '' OR action = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("audit: list: %w", err)
	}
	defer rows.Close()

	var out []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.Action, &e.TargetType, &e.TargetID, &e.RequestID, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("audit: scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
