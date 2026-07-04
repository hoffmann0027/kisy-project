// Package search maintains a denormalized full-text index of messages and
// answers scoped search queries. Results are limited to chats the actor
// actually participates in, so search never leaks messages across clearances.
package search

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Result is one message match returned to the client.
type Result struct {
	MessageID  uuid.UUID `json:"messageId"`
	ChatType   string    `json:"chatType"`
	ChatID     uuid.UUID `json:"chatId"`
	SenderID   uuid.UUID `json:"senderId"`
	SenderName string    `json:"senderName"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewService(pool *pgxpool.Pool, log *slog.Logger) *Service {
	return &Service{pool: pool, log: log}
}

// IndexMessage upserts a message's searchable content. Called from the message
// lifecycle; best-effort, so a failure logs but never blocks messaging.
func (s *Service) IndexMessage(ctx context.Context, messageID uuid.UUID, content string) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO search_index (entity_type, entity_id, content)
		VALUES ('message', $1, $2)
		ON CONFLICT (entity_type, entity_id)
		DO UPDATE SET content = EXCLUDED.content`,
		messageID, content)
	if err != nil {
		s.log.Warn("search index upsert failed", "error", err)
	}
}

// RemoveMessage drops a message from the index (on delete).
func (s *Service) RemoveMessage(ctx context.Context, messageID uuid.UUID) {
	if _, err := s.pool.Exec(ctx, `DELETE FROM search_index WHERE entity_type = 'message' AND entity_id = $1`, messageID); err != nil {
		s.log.Warn("search index delete failed", "error", err)
	}
}

// Search returns messages matching the query that the actor may see. Scoping
// is done in SQL against the actor's private-chat participation and group
// membership.
func (s *Service) Search(ctx context.Context, actorID uuid.UUID, query string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []Result{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 25
	}

	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.chat_type, m.chat_id, m.sender_id, u.display_name, m.text, m.created_at
		FROM search_index si
		JOIN messages m ON m.id = si.entity_id
		JOIN users u ON u.id = m.sender_id
		WHERE si.entity_type = 'message'
		  AND si.search_vector @@ plainto_tsquery('russian', $2)
		  AND m.is_deleted = false
		  AND (
		    (m.chat_type = 'private' AND m.chat_id IN (
		       SELECT id FROM private_chats WHERE user_a_id = $1 OR user_b_id = $1))
		    OR
		    (m.chat_type = 'group' AND m.chat_id IN (
		       SELECT group_id FROM group_members WHERE user_id = $1))
		  )
		ORDER BY ts_rank(si.search_vector, plainto_tsquery('russian', $2)) DESC, m.created_at DESC
		LIMIT $3`,
		actorID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search: query: %w", err)
	}
	defer rows.Close()

	out := make([]Result, 0, limit)
	for rows.Next() {
		var r Result
		var text *string
		if err := rows.Scan(&r.MessageID, &r.ChatType, &r.ChatID, &r.SenderID, &r.SenderName, &text, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("search: scan: %w", err)
		}
		if text != nil {
			r.Text = *text
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
