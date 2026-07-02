// Package readstate tracks each user's read position per chat and derives
// unread counters, implementing the "update unread counters" step of the
// message lifecycle (docs/spec/07-business-logic.md).
package readstate

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

// ChatAuthorizer confirms the actor may access a chat before their read
// position is stored. Injected to avoid import cycles.
type ChatAuthorizer func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, actorLevel int) error

// Repository is the persistence port for read state.
type Repository interface {
	MarkRead(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID, messageID uuid.UUID) error
	// UnreadForChats returns unread counts for the given chats of one type.
	// Unread = messages from other users, not deleted, created after the
	// user's last-read timestamp (or all such messages if never read).
	UnreadForChats(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatIDs []uuid.UUID) (map[uuid.UUID]int, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) MarkRead(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID, messageID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO chat_read_state (user_id, chat_type, chat_id, last_read_at, last_read_message_id)
		VALUES ($1, $2, $3, now(), $4)
		ON CONFLICT (user_id, chat_type, chat_id)
		DO UPDATE SET last_read_at = now(), last_read_message_id = EXCLUDED.last_read_message_id`,
		userID, chatType, chatID, messageID)
	if err != nil {
		return fmt.Errorf("readstate: mark read: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UnreadForChats(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(chatIDs))
	if len(chatIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT m.chat_id, count(*)
		FROM messages m
		LEFT JOIN chat_read_state s
		  ON s.user_id = $1 AND s.chat_type = m.chat_type AND s.chat_id = m.chat_id
		WHERE m.chat_type = $2
		  AND m.chat_id = ANY($3)
		  AND m.sender_id <> $1
		  AND m.is_deleted = false
		  AND m.created_at > COALESCE(s.last_read_at, 'epoch'::timestamptz)
		GROUP BY m.chat_id`,
		userID, chatType, chatIDs)
	if err != nil {
		return nil, fmt.Errorf("readstate: unread counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("readstate: scan: %w", err)
		}
		out[id] = n
	}
	return out, rows.Err()
}

// Actor identifies the acting user.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

type Service struct {
	pool      *pgxpool.Pool
	repo      Repository
	authorize ChatAuthorizer
}

func NewService(pool *pgxpool.Pool, repo Repository, authorize ChatAuthorizer) *Service {
	return &Service{pool: pool, repo: repo, authorize: authorize}
}

// MarkRead stores the actor's read position after authorizing chat access.
func (s *Service) MarkRead(ctx context.Context, chatType string, chatID, messageID uuid.UUID, actor Actor) error {
	if err := s.authorize(ctx, chatType, chatID, actor.UserID, actor.RoleLevel); err != nil {
		return err
	}
	return s.repo.MarkRead(ctx, s.pool, actor.UserID, chatType, chatID, messageID)
}

// UnreadForPrivateChats returns unread counts keyed by chat id.
func (s *Service) UnreadForPrivateChats(ctx context.Context, userID uuid.UUID, chatIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	return s.repo.UnreadForChats(ctx, s.pool, userID, "private", chatIDs)
}

// PersistRead is a fire-and-forget hook for the WebSocket read receipt; it
// swallows the (already authorized upstream) error to a boolean.
func (s *Service) PersistRead(ctx context.Context, userID uuid.UUID, chatType string, chatID, messageID uuid.UUID) {
	_ = s.repo.MarkRead(ctx, s.pool, userID, chatType, chatID, messageID)
}
