// Package notifications stores per-user notifications and raises them for
// @mentions in messages (docs/spec/03-backend-architecture.md: "Audit"
// aside, and 09 notification.created). Unread *messages* are tracked by
// the readstate package; this package handles discrete events.
package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

// Notification types.
const TypeMention = "mention"

// Notification mirrors a notifications row.
type Notification struct {
	ID        uuid.UUID       `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	IsRead    bool            `json:"isRead"`
	CreatedAt time.Time       `json:"createdAt"`
}

// Repository is the persistence port for notifications.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, userID uuid.UUID, typ string, payload []byte) error
	ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, q db.DBTX, userID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, q db.DBTX, userID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, q db.DBTX, userID uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, userID uuid.UUID, typ string, payload []byte) error {
	_, err := q.Exec(ctx, `
		INSERT INTO notifications (user_id, type, payload) VALUES ($1, $2, $3)`,
		userID, typ, payload)
	if err != nil {
		return fmt.Errorf("notifications: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID, limit int) ([]Notification, error) {
	rows, err := q.Query(ctx, `
		SELECT id, type, payload, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("notifications: list: %w", err)
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Payload, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("notifications: scan: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UnreadCount(ctx context.Context, q db.DBTX, userID uuid.UUID) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("notifications: unread count: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) MarkRead(ctx context.Context, q db.DBTX, userID, id uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE notifications SET is_read = true, read_at = now()
		WHERE user_id = $1 AND id = $2 AND is_read = false`, userID, id)
	if err != nil {
		return fmt.Errorf("notifications: mark read: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, q db.DBTX, userID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE notifications SET is_read = true, read_at = now()
		WHERE user_id = $1 AND is_read = false`, userID)
	if err != nil {
		return fmt.Errorf("notifications: mark all read: %w", err)
	}
	return nil
}

// AddMention records a resolved @mention on a message (satisfies
// MentionSink). Duplicate mentions are ignored.
func (r *PostgresRepository) AddMention(ctx context.Context, q db.DBTX, messageID, userID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO message_mentions (message_id, mentioned_user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, messageID, userID)
	if err != nil {
		return fmt.Errorf("notifications: add mention: %w", err)
	}
	return nil
}

// RecipientResolver returns the user IDs that can receive events for a
// chat. UsernameResolver maps a username to a user ID (ok=false if
// unknown/inactive). WSPublisher pushes a live notification.created event.
type RecipientResolver func(ctx context.Context, chatType string, chatID uuid.UUID) ([]uuid.UUID, error)
type UsernameResolver func(ctx context.Context, username string) (uuid.UUID, bool)
type WSPublisher interface {
	PublishNotification(userID uuid.UUID, data any)
}

// MentionSink persists resolved @mentions on a message.
type MentionSink interface {
	AddMention(ctx context.Context, q db.DBTX, messageID, userID uuid.UUID) error
}

// Pusher sends a Web Push notification to a user's browsers. Injected to avoid
// a notifications→push import cycle; nil disables push.
type Pusher interface {
	Notify(ctx context.Context, userID uuid.UUID, title, body, url string)
}

// Preferences gates notifications by the recipient's mute state and group
// notification mode (stage G). Injected to avoid a notifications→notifprefs
// import cycle; nil means "notify everyone" (pre-stage-G behavior).
type Preferences interface {
	// MutedUsers returns which of userIDs have the chat muted right now.
	MutedUsers(ctx context.Context, chatType string, chatID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]struct{}, error)
	// SettingsFor batch-loads each user's settings (sound/preview/groupMode).
	SettingsFor(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]NotifSettings, error)
}

// NotifSettings mirrors the fields the pipeline needs from a user's prefs.
type NotifSettings struct {
	Sound     bool
	Preview   bool
	GroupMode string // "all" | "mentions_only" | "none"
}

type Service struct {
	pool       *pgxpool.Pool
	repo       Repository
	recipients RecipientResolver
	usernames  UsernameResolver
	mentions   MentionSink
	ws         WSPublisher
	pusher     Pusher
	prefs      Preferences
}

// SetPusher wires Web Push delivery for new notifications.
func (s *Service) SetPusher(p Pusher) { s.pusher = p }

// SetPreferences wires the mute/notification-mode gate.
func (s *Service) SetPreferences(p Preferences) { s.prefs = p }

func NewService(pool *pgxpool.Pool, repo Repository, recipients RecipientResolver, usernames UsernameResolver, mentions MentionSink, wsPub WSPublisher) *Service {
	return &Service{pool: pool, repo: repo, recipients: recipients, usernames: usernames, mentions: mentions, ws: wsPub}
}

// List returns the actor's most recent notifications and the unread count.
func (s *Service) List(ctx context.Context, userID uuid.UUID, limit int) ([]Notification, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	list, err := s.repo.ListForUser(ctx, s.pool, userID, limit)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.UnreadCount(ctx, s.pool, userID)
	if err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []Notification{}
	}
	return list, unread, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.MarkRead(ctx, s.pool, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, s.pool, userID)
}
