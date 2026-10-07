// Package readstate tracks each user's read position per chat and derives
// unread counters, implementing the "update unread counters" step of the
// message lifecycle (docs/spec/07-business-logic.md).
package readstate

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	// OthersLastRead returns, per private chat, the last-read timestamp of
	// the *other* participant. Because a private chat has exactly two users,
	// "user_id <> self" uniquely identifies the counterpart, so no explicit
	// participant list is needed. Chats the other user never read are absent.
	OthersLastRead(ctx context.Context, q db.DBTX, self uuid.UUID, chatIDs []uuid.UUID) (map[uuid.UUID]time.Time, error)
	// GroupReads returns every member's last-read timestamp for a group chat,
	// keyed by user id. Members who never opened the chat are absent.
	GroupReads(ctx context.Context, q db.DBTX, chatID uuid.UUID) (map[uuid.UUID]time.Time, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) MarkRead(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID, messageID uuid.UUID) error {
	// The message has to be one of this chat's. Any id used to be stored —
	// one from somebody else's chat included — and then sent to this chat's
	// members as the reader's position (audit A-37). The casts are needed:
	// the same parameter appears as a value and in a comparison, and Postgres
	// will not guess one type for both.
	tag, err := q.Exec(ctx, `
		INSERT INTO chat_read_state (user_id, chat_type, chat_id, last_read_at, last_read_message_id)
		SELECT $1::uuid, $2::text, $3::uuid, now(), $4::uuid
		WHERE EXISTS (SELECT 1 FROM messages WHERE id = $4::uuid AND chat_type = $2::text AND chat_id = $3::uuid)
		ON CONFLICT (user_id, chat_type, chat_id)
		DO UPDATE SET last_read_at = now(), last_read_message_id = EXCLUDED.last_read_message_id`,
		userID, chatType, chatID, messageID)
	if err != nil {
		return fmt.Errorf("readstate: mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMessageNotInChat
	}
	return nil
}

// ErrMessageNotInChat: the read position names a message of another chat, or
// none at all.
var ErrMessageNotInChat = errors.New("readstate: message is not in this chat")

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

func (r *PostgresRepository) OthersLastRead(ctx context.Context, q db.DBTX, self uuid.UUID, chatIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	out := make(map[uuid.UUID]time.Time, len(chatIDs))
	if len(chatIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT chat_id, last_read_at
		FROM chat_read_state
		WHERE chat_type = 'private'
		  AND chat_id = ANY($1)
		  AND user_id <> $2`,
		chatIDs, self)
	if err != nil {
		return nil, fmt.Errorf("readstate: others last read: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, fmt.Errorf("readstate: scan: %w", err)
		}
		out[id] = at
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GroupReads(ctx context.Context, q db.DBTX, chatID uuid.UUID) (map[uuid.UUID]time.Time, error) {
	rows, err := q.Query(ctx, `
		SELECT user_id, last_read_at FROM chat_read_state
		WHERE chat_type = 'group' AND chat_id = $1`, chatID)
	if err != nil {
		return nil, fmt.Errorf("readstate: group reads: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]time.Time)
	for rows.Next() {
		var uid uuid.UUID
		var at time.Time
		if err := rows.Scan(&uid, &at); err != nil {
			return nil, fmt.Errorf("readstate: scan group read: %w", err)
		}
		out[uid] = at
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

// OthersLastReadPrivate returns, per private chat, when the counterpart last
// read it — used to render read receipts (ticks) on the actor's own messages.
func (s *Service) OthersLastReadPrivate(ctx context.Context, userID uuid.UUID, chatIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	return s.repo.OthersLastRead(ctx, s.pool, userID, chatIDs)
}

// GroupReads returns each member's last-read time for a group chat, used to
// compute the per-message "read by N of M" counter.
func (s *Service) GroupReads(ctx context.Context, chatID uuid.UUID) (map[uuid.UUID]time.Time, error) {
	return s.repo.GroupReads(ctx, s.pool, chatID)
}

// PersistRead is the WebSocket read receipt's hook. It reports whether the
// position was stored: the hub announces a read to the chat only when it was,
// so a message id from elsewhere is never broadcast (audit A-37).
func (s *Service) PersistRead(ctx context.Context, userID uuid.UUID, chatType string, chatID, messageID uuid.UUID) bool {
	return s.repo.MarkRead(ctx, s.pool, userID, chatType, chatID, messageID) == nil
}
