// Package reactions implements emoji reactions on messages: persistence,
// REST add/remove and real-time broadcast. Access is authorized through
// the messages service so the chat visibility rules are not duplicated.
package reactions

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/db"
)

var (
	ErrInvalidEmoji = errors.New("reactions: invalid emoji")
	ErrNotFound     = errors.New("reactions: message not found")
)

// MaxEmojiLength bounds a reaction token (a short emoji or shortcode).
const MaxEmojiLength = 32

// Repository is the persistence port for reactions.
type Repository interface {
	Add(ctx context.Context, q db.DBTX, messageID, userID uuid.UUID, emoji string) error
	Remove(ctx context.Context, q db.DBTX, messageID, userID uuid.UUID, emoji string) error
	SummariesFor(ctx context.Context, q db.DBTX, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]messages.ReactionSummary, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Add(ctx context.Context, q db.DBTX, messageID, userID uuid.UUID, emoji string) error {
	// Idempotent: re-reacting with the same emoji is a no-op.
	_, err := q.Exec(ctx, `
		INSERT INTO reactions (message_id, user_id, emoji) VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING`,
		messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("reactions: add: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Remove(ctx context.Context, q db.DBTX, messageID, userID uuid.UUID, emoji string) error {
	_, err := q.Exec(ctx, `
		DELETE FROM reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`,
		messageID, userID, emoji)
	if err != nil {
		return fmt.Errorf("reactions: remove: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SummariesFor(ctx context.Context, q db.DBTX, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]messages.ReactionSummary, error) {
	rows, err := q.Query(ctx, `
		SELECT message_id, emoji, count(*) AS n, bool_or(user_id = $2) AS reacted
		FROM reactions
		WHERE message_id = ANY($1)
		GROUP BY message_id, emoji
		ORDER BY n DESC, emoji ASC`,
		messageIDs, viewerID)
	if err != nil {
		return nil, fmt.Errorf("reactions: summaries: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]messages.ReactionSummary)
	for rows.Next() {
		var mid uuid.UUID
		var s messages.ReactionSummary
		if err := rows.Scan(&mid, &s.Emoji, &s.Count, &s.Reacted); err != nil {
			return nil, fmt.Errorf("reactions: scan: %w", err)
		}
		out[mid] = append(out[mid], s)
	}
	return out, rows.Err()
}

// MessageAccess authorizes and locates a message; satisfied by
// *messages.Service.
type MessageAccess interface {
	ResolveAccessible(ctx context.Context, messageID uuid.UUID, actor messages.ActorMeta) (string, uuid.UUID, error)
}

// Publisher broadcasts a reaction change to a chat's participants.
type Publisher interface {
	PublishReaction(chatType string, chatID, messageID, userID uuid.UUID, emoji string, added bool)
}

// Actor identifies the reacting user.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

type Service struct {
	pool   *pgxpool.Pool
	repo   Repository
	access MessageAccess
	pub    Publisher
}

func NewService(pool *pgxpool.Pool, repo Repository, access MessageAccess) *Service {
	return &Service{pool: pool, repo: repo, access: access}
}

func (s *Service) SetPublisher(p Publisher) { s.pub = p }

// Loader exposes the batched summary query as a messages.ReactionLoader.
func (s *Service) Loader(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]messages.ReactionSummary, error) {
	return s.repo.SummariesFor(ctx, s.pool, messageIDs, viewerID)
}

func (s *Service) toMessagesActor(a Actor) messages.ActorMeta {
	return messages.ActorMeta{UserID: a.UserID, RoleLevel: a.RoleLevel}
}

// Add records a reaction after confirming the actor may access the message.
func (s *Service) Add(ctx context.Context, messageID uuid.UUID, emoji string, actor Actor) error {
	if emoji == "" || len(emoji) > MaxEmojiLength {
		return ErrInvalidEmoji
	}
	chatType, chatID, err := s.access.ResolveAccessible(ctx, messageID, s.toMessagesActor(actor))
	if errors.Is(err, messages.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := s.repo.Add(ctx, s.pool, messageID, actor.UserID, emoji); err != nil {
		return err
	}
	if s.pub != nil {
		s.pub.PublishReaction(chatType, chatID, messageID, actor.UserID, emoji, true)
	}
	return nil
}

// Remove deletes the actor's reaction.
func (s *Service) Remove(ctx context.Context, messageID uuid.UUID, emoji string, actor Actor) error {
	if emoji == "" || len(emoji) > MaxEmojiLength {
		return ErrInvalidEmoji
	}
	chatType, chatID, err := s.access.ResolveAccessible(ctx, messageID, s.toMessagesActor(actor))
	if errors.Is(err, messages.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := s.repo.Remove(ctx, s.pool, messageID, actor.UserID, emoji); err != nil {
		return err
	}
	if s.pub != nil {
		s.pub.PublishReaction(chatType, chatID, messageID, actor.UserID, emoji, false)
	}
	return nil
}
