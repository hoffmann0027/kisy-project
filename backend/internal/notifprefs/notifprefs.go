// Package notifprefs owns per-user notification preferences: chat mutes and
// notification settings (stage G). It is queried by the notifications
// pipeline to decide whether to push, and exposes HTTP endpoints for the
// client to manage its own preferences.
package notifprefs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

var ErrValidation = errors.New("notifprefs: validation failed")

// Group notification modes.
const (
	GroupAll          = "all"
	GroupMentionsOnly = "mentions_only"
	GroupNone         = "none"
)

// Settings is a user's notification preferences. Absent rows use these
// defaults (sound+preview on, groups notify on everything).
type Settings struct {
	Sound     bool   `json:"sound"`
	Preview   bool   `json:"preview"`
	GroupMode string `json:"groupMode"`
}

func DefaultSettings() Settings {
	return Settings{Sound: true, Preview: true, GroupMode: GroupAll}
}

// Repository is the persistence port for mutes and settings.
type Repository interface {
	SetMute(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID, until *time.Time) error
	Unmute(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error
	// MutedChats returns the subset of userIDs for whom (chatType, chatID) is
	// currently muted (muted_until NULL or in the future).
	MutedUsers(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, userIDs []uuid.UUID, now time.Time) (map[uuid.UUID]struct{}, error)
	// ListMutes returns the actor's active mutes for hydrating the chat list.
	ListMutes(ctx context.Context, q db.DBTX, userID uuid.UUID, now time.Time) ([]Mute, error)
	GetSettings(ctx context.Context, q db.DBTX, userID uuid.UUID) (Settings, error)
	SettingsFor(ctx context.Context, q db.DBTX, userIDs []uuid.UUID) (map[uuid.UUID]Settings, error)
	UpsertSettings(ctx context.Context, q db.DBTX, userID uuid.UUID, s Settings) error
}

// Mute is one active mute entry (for the chat list).
type Mute struct {
	ChatType   string     `json:"chatType"`
	ChatID     uuid.UUID  `json:"chatId"`
	MutedUntil *time.Time `json:"mutedUntil"`
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) SetMute(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID, until *time.Time) error {
	_, err := q.Exec(ctx, `
		INSERT INTO chat_mutes (user_id, chat_type, chat_id, muted_until)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, chat_type, chat_id) DO UPDATE SET muted_until = EXCLUDED.muted_until`,
		userID, chatType, chatID, until)
	if err != nil {
		return fmt.Errorf("notifprefs: set mute: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Unmute(ctx context.Context, q db.DBTX, userID uuid.UUID, chatType string, chatID uuid.UUID) error {
	_, err := q.Exec(ctx, `DELETE FROM chat_mutes WHERE user_id = $1 AND chat_type = $2 AND chat_id = $3`,
		userID, chatType, chatID)
	if err != nil {
		return fmt.Errorf("notifprefs: unmute: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MutedUsers(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, userIDs []uuid.UUID, now time.Time) (map[uuid.UUID]struct{}, error) {
	out := make(map[uuid.UUID]struct{})
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT user_id FROM chat_mutes
		WHERE chat_type = $1 AND chat_id = $2 AND user_id = ANY($3)
		  AND (muted_until IS NULL OR muted_until > $4)`,
		chatType, chatID, userIDs, now)
	if err != nil {
		return nil, fmt.Errorf("notifprefs: muted users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("notifprefs: scan muted: %w", err)
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListMutes(ctx context.Context, q db.DBTX, userID uuid.UUID, now time.Time) ([]Mute, error) {
	rows, err := q.Query(ctx, `
		SELECT chat_type, chat_id, muted_until FROM chat_mutes
		WHERE user_id = $1 AND (muted_until IS NULL OR muted_until > $2)`,
		userID, now)
	if err != nil {
		return nil, fmt.Errorf("notifprefs: list mutes: %w", err)
	}
	defer rows.Close()
	var out []Mute
	for rows.Next() {
		var m Mute
		if err := rows.Scan(&m.ChatType, &m.ChatID, &m.MutedUntil); err != nil {
			return nil, fmt.Errorf("notifprefs: scan mute: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetSettings(ctx context.Context, q db.DBTX, userID uuid.UUID) (Settings, error) {
	s := DefaultSettings()
	err := q.QueryRow(ctx, `SELECT sound, preview, group_mode FROM notification_settings WHERE user_id = $1`, userID).
		Scan(&s.Sound, &s.Preview, &s.GroupMode)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("notifprefs: get settings: %w", err)
	}
	return s, nil
}

func (r *PostgresRepository) SettingsFor(ctx context.Context, q db.DBTX, userIDs []uuid.UUID) (map[uuid.UUID]Settings, error) {
	out := make(map[uuid.UUID]Settings, len(userIDs))
	for _, id := range userIDs {
		out[id] = DefaultSettings()
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT user_id, sound, preview, group_mode FROM notification_settings WHERE user_id = ANY($1)`, userIDs)
	if err != nil {
		return nil, fmt.Errorf("notifprefs: settings for: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var s Settings
		if err := rows.Scan(&id, &s.Sound, &s.Preview, &s.GroupMode); err != nil {
			return nil, fmt.Errorf("notifprefs: scan settings: %w", err)
		}
		out[id] = s
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpsertSettings(ctx context.Context, q db.DBTX, userID uuid.UUID, s Settings) error {
	_, err := q.Exec(ctx, `
		INSERT INTO notification_settings (user_id, sound, preview, group_mode, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (user_id) DO UPDATE
		SET sound = EXCLUDED.sound, preview = EXCLUDED.preview, group_mode = EXCLUDED.group_mode, updated_at = now()`,
		userID, s.Sound, s.Preview, s.GroupMode)
	if err != nil {
		return fmt.Errorf("notifprefs: upsert settings: %w", err)
	}
	return nil
}

// Service exposes preference operations and the gate used by the
// notifications pipeline.
type Service struct {
	pool *pgxpool.Pool
	repo Repository
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// Mute mutes a chat for the actor until `until` (nil = forever).
func (s *Service) Mute(ctx context.Context, userID uuid.UUID, chatType string, chatID uuid.UUID, until *time.Time) error {
	if chatType != "private" && chatType != "group" {
		return ErrValidation
	}
	if until != nil && until.Before(time.Now()) {
		return ErrValidation
	}
	return s.repo.SetMute(ctx, s.pool, userID, chatType, chatID, until)
}

func (s *Service) Unmute(ctx context.Context, userID uuid.UUID, chatType string, chatID uuid.UUID) error {
	return s.repo.Unmute(ctx, s.pool, userID, chatType, chatID)
}

func (s *Service) ListMutes(ctx context.Context, userID uuid.UUID) ([]Mute, error) {
	list, err := s.repo.ListMutes(ctx, s.pool, userID, time.Now())
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Mute{}
	}
	return list, nil
}

func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) (Settings, error) {
	return s.repo.GetSettings(ctx, s.pool, userID)
}

func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, in Settings) (Settings, error) {
	switch in.GroupMode {
	case GroupAll, GroupMentionsOnly, GroupNone:
	default:
		return Settings{}, ErrValidation
	}
	if err := s.repo.UpsertSettings(ctx, s.pool, userID, in); err != nil {
		return Settings{}, err
	}
	return in, nil
}

// MutedUsers returns which of userIDs have (chatType, chatID) muted now —
// the gate the notifications pipeline calls per message.
func (s *Service) MutedUsers(ctx context.Context, chatType string, chatID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	return s.repo.MutedUsers(ctx, s.pool, chatType, chatID, userIDs, time.Now())
}

// SettingsFor batch-loads settings for several recipients.
func (s *Service) SettingsFor(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Settings, error) {
	return s.repo.SettingsFor(ctx, s.pool, userIDs)
}
