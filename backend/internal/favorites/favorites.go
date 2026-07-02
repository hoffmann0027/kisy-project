// Package favorites implements favorite and pinned chats
// (docs/spec/02-frontend-ux.md: "pinned chats, favorites"). A user may
// only favorite a chat they are allowed to access.
package favorites

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

// ChatAuthorizer confirms the actor may access a chat. Injected to avoid
// import cycles.
type ChatAuthorizer func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, actorLevel int) error

// Favorite mirrors a favorites row for one user.
type Favorite struct {
	ChatType    string    `json:"chatType"`
	ChatID      uuid.UUID `json:"chatId"`
	IsPinned    bool      `json:"isPinned"`
	PinnedOrder *int      `json:"pinnedOrder"`
}

// Repository is the persistence port for favorites.
type Repository interface {
	Upsert(ctx context.Context, q db.DBTX, userID uuid.UUID, f Favorite) error
	Remove(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error
	ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]Favorite, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Upsert(ctx context.Context, q db.DBTX, userID uuid.UUID, f Favorite) error {
	_, err := q.Exec(ctx, `
		INSERT INTO favorites (user_id, chat_type, chat_id, is_pinned, pinned_order)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, chat_type, chat_id)
		DO UPDATE SET is_pinned = EXCLUDED.is_pinned, pinned_order = EXCLUDED.pinned_order`,
		userID, f.ChatType, f.ChatID, f.IsPinned, f.PinnedOrder)
	if err != nil {
		return fmt.Errorf("favorites: upsert: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Remove(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		DELETE FROM favorites WHERE user_id = $1 AND chat_type = $2 AND chat_id = $3`,
		userID, chatType, chatID)
	if err != nil {
		return fmt.Errorf("favorites: remove: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]Favorite, error) {
	rows, err := q.Query(ctx, `
		SELECT chat_type, chat_id, is_pinned, pinned_order
		FROM favorites
		WHERE user_id = $1
		ORDER BY is_pinned DESC, pinned_order ASC NULLS LAST, created_at DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("favorites: list: %w", err)
	}
	defer rows.Close()

	var out []Favorite
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(&f.ChatType, &f.ChatID, &f.IsPinned, &f.PinnedOrder); err != nil {
			return nil, fmt.Errorf("favorites: scan: %w", err)
		}
		out = append(out, f)
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

// Set adds or updates a favorite/pin after authorizing chat access.
func (s *Service) Set(ctx context.Context, f Favorite, actor Actor) error {
	if err := s.authorize(ctx, f.ChatType, f.ChatID, actor.UserID, actor.RoleLevel); err != nil {
		return err
	}
	return s.repo.Upsert(ctx, s.pool, actor.UserID, f)
}

// Remove drops a favorite. No authorization is needed to forget a chat.
func (s *Service) Remove(ctx context.Context, chatType string, chatID uuid.UUID, actor Actor) error {
	return s.repo.Remove(ctx, s.pool, actor.UserID, chatType, chatID)
}

// List returns the actor's favorites, pinned first.
func (s *Service) List(ctx context.Context, actor Actor) ([]Favorite, error) {
	return s.repo.ListForUser(ctx, s.pool, actor.UserID)
}
