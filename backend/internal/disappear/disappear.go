// Package disappear implements disappearing messages (UPD3 stage J): a
// chat-wide default TTL for new messages plus the reaper that HARD-deletes
// expired rows — text, ciphertext and attachment bytes all leave the
// database (cascade), and every client is told via message.deleted so it
// purges its local plaintext cache too (docs/security.md).
//
// The timer itself (expires_at) is metadata, like reply_to: in E2EE chats
// the server sees when a message dies but never what it said.
package disappear

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/db"
)

var (
	ErrValidation = errors.New("disappear: validation failed")
	// ErrNotFound masks both "no such chat" and "no access" (404, never 403).
	ErrNotFound = errors.New("disappear: not found")
)

// TTL bounds: 5s (useful for demos/tests) to 1 year.
const (
	MinTTLSeconds = 5
	MaxTTLSeconds = 365 * 24 * 3600
)

// ActionDisappearingSet is the audit action for enabling/changing/disabling
// a chat's timer. Metadata carries the TTL, never any content.
const ActionDisappearingSet = "chat.disappearing_set"

// ChatAuthorizer confirms the actor may access a chat (masked as
// ErrNotFound by the service on failure).
type ChatAuthorizer func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, actorLevel int) error

// Publisher fans the expiry out to connected clients. Satisfied by
// ws.Publisher; the event is message.deleted with expired=true so clients
// remove the bubble entirely (no tombstone) and purge local caches.
type Publisher interface {
	PublishMessageExpired(chatType string, chatID, messageID uuid.UUID)
}

// Indexer removes reaped messages from the full-text index (search_index
// has no FK on messages, so the cascade does not cover it).
type Indexer interface {
	RemoveMessage(ctx context.Context, messageID uuid.UUID)
}

// Actor identifies the acting user.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

// Setting is a chat's disappearing-message default.
type Setting struct {
	TTLSeconds *int64     `json:"ttlSeconds"`
	SetBy      *uuid.UUID `json:"setBy,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}

// Repository is the persistence port.
type Repository interface {
	GetTTL(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID) (Setting, error)
	SetTTL(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, ttlSeconds int64, setBy uuid.UUID) error
	ClearTTL(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID) error
	// ClaimExpired locks up to limit expired message rows (FOR UPDATE SKIP
	// LOCKED) and returns their coordinates for the post-commit fan-out.
	ClaimExpired(ctx context.Context, q db.DBTX, now time.Time, limit int) ([]ExpiredRef, error)
	// DeleteMessages hard-deletes the rows; attachments/reactions/mentions
	// go with them via ON DELETE CASCADE.
	DeleteMessages(ctx context.Context, q db.DBTX, ids []uuid.UUID) error
}

// ExpiredRef locates one reaped message for the client fan-out.
type ExpiredRef struct {
	ID       uuid.UUID
	ChatType string
	ChatID   uuid.UUID
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) GetTTL(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID) (Setting, error) {
	var s Setting
	err := q.QueryRow(ctx, `
		SELECT ttl_seconds, set_by, updated_at FROM chat_disappear_settings
		WHERE chat_type = $1 AND chat_id = $2`, chatType, chatID).
		Scan(&s.TTLSeconds, &s.SetBy, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Setting{}, nil
	}
	if err != nil {
		return Setting{}, fmt.Errorf("disappear: get ttl: %w", err)
	}
	return s, nil
}

func (r *PostgresRepository) SetTTL(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID, ttlSeconds int64, setBy uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO chat_disappear_settings (chat_type, chat_id, ttl_seconds, set_by, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (chat_type, chat_id) DO UPDATE
		SET ttl_seconds = EXCLUDED.ttl_seconds, set_by = EXCLUDED.set_by, updated_at = now()`,
		chatType, chatID, ttlSeconds, setBy)
	if err != nil {
		return fmt.Errorf("disappear: set ttl: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ClearTTL(ctx context.Context, q db.DBTX, chatType string, chatID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		DELETE FROM chat_disappear_settings WHERE chat_type = $1 AND chat_id = $2`, chatType, chatID)
	if err != nil {
		return fmt.Errorf("disappear: clear ttl: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ClaimExpired(ctx context.Context, q db.DBTX, now time.Time, limit int) ([]ExpiredRef, error) {
	rows, err := q.Query(ctx, `
		SELECT id, chat_type, chat_id FROM messages
		WHERE expires_at IS NOT NULL AND expires_at <= $1
		ORDER BY expires_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("disappear: claim expired: %w", err)
	}
	defer rows.Close()
	var out []ExpiredRef
	for rows.Next() {
		var ref ExpiredRef
		if err := rows.Scan(&ref.ID, &ref.ChatType, &ref.ChatID); err != nil {
			return nil, fmt.Errorf("disappear: scan expired: %w", err)
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) DeleteMessages(ctx context.Context, q db.DBTX, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `DELETE FROM messages WHERE id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("disappear: delete: %w", err)
	}
	return nil
}

// Service owns the chat timer setting and the reaper.
type Service struct {
	pool      *pgxpool.Pool
	repo      Repository
	authorize ChatAuthorizer
	audit     audit.Recorder
	pub       Publisher
	indexer   Indexer
}

func NewService(pool *pgxpool.Pool, repo Repository, authorize ChatAuthorizer, rec audit.Recorder) *Service {
	return &Service{pool: pool, repo: repo, authorize: authorize, audit: rec}
}

// SetPublisher wires the real-time fan-out (constructed together at startup).
func (s *Service) SetPublisher(p Publisher) { s.pub = p }

// SetIndexer wires full-text index cleanup for reaped messages.
func (s *Service) SetIndexer(i Indexer) { s.indexer = i }

// TTLFor satisfies messages.DisappearTTL: the chat's default timer for new
// messages (nil = off). No access check — it is only called from the send
// path, which has already authorized the actor.
func (s *Service) TTLFor(ctx context.Context, chatType string, chatID uuid.UUID) (*int64, error) {
	setting, err := s.repo.GetTTL(ctx, s.pool, chatType, chatID)
	if err != nil {
		return nil, err
	}
	return setting.TTLSeconds, nil
}

// Get returns the chat's timer for a member (masked 404 otherwise).
func (s *Service) Get(ctx context.Context, chatType string, chatID uuid.UUID, actor Actor) (Setting, error) {
	if chatType != "private" && chatType != "group" {
		return Setting{}, ErrValidation
	}
	if err := s.authorize(ctx, chatType, chatID, actor.UserID, actor.RoleLevel); err != nil {
		return Setting{}, ErrNotFound
	}
	return s.repo.GetTTL(ctx, s.pool, chatType, chatID)
}

// Set enables, changes or disables (ttlSeconds == nil/0) the chat's default
// timer. Any member may change it — the change is visible to everyone and
// audited (TTL only, never content).
func (s *Service) Set(ctx context.Context, chatType string, chatID uuid.UUID, ttlSeconds *int64, actor Actor) (Setting, error) {
	if chatType != "private" && chatType != "group" {
		return Setting{}, ErrValidation
	}
	enabled := ttlSeconds != nil && *ttlSeconds > 0
	if enabled && (*ttlSeconds < MinTTLSeconds || *ttlSeconds > MaxTTLSeconds) {
		return Setting{}, ErrValidation
	}
	if err := s.authorize(ctx, chatType, chatID, actor.UserID, actor.RoleLevel); err != nil {
		return Setting{}, ErrNotFound
	}

	if enabled {
		if err := s.repo.SetTTL(ctx, s.pool, chatType, chatID, *ttlSeconds, actor.UserID); err != nil {
			return Setting{}, err
		}
	} else {
		if err := s.repo.ClearTTL(ctx, s.pool, chatType, chatID); err != nil {
			return Setting{}, err
		}
	}

	if s.audit != nil {
		meta := map[string]any{"chatType": chatType, "chatId": chatID.String()}
		if enabled {
			meta["ttlSeconds"] = *ttlSeconds
		} else {
			meta["ttlSeconds"] = nil
		}
		_ = s.audit.Record(ctx, s.pool, audit.Event{
			ActorID:    &actor.UserID,
			Action:     ActionDisappearingSet,
			TargetType: "chat",
			TargetID:   &chatID,
			Metadata:   meta,
		})
	}
	return s.repo.GetTTL(ctx, s.pool, chatType, chatID)
}

// ProcessExpired hard-deletes every message past its expires_at and returns
// how many were reaped. Deletion happens in one transaction (rows +
// cascaded attachments/reactions); the client fan-out and search-index
// cleanup run after commit. This is a SYSTEM deletion — it does not widen
// any user's ability to delete others' messages.
func (s *Service) ProcessExpired(ctx context.Context, now time.Time, batch int) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("disappear: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	refs, err := s.repo.ClaimExpired(ctx, tx, now, batch)
	if err != nil {
		return 0, err
	}
	if len(refs) == 0 {
		return 0, nil
	}
	ids := make([]uuid.UUID, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
	}
	if err := s.repo.DeleteMessages(ctx, tx, ids); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("disappear: commit: %w", err)
	}

	for _, ref := range refs {
		if s.pub != nil {
			s.pub.PublishMessageExpired(ref.ChatType, ref.ChatID, ref.ID)
		}
		if s.indexer != nil {
			s.indexer.RemoveMessage(ctx, ref.ID)
		}
	}
	return len(refs), nil
}

// StartReaper runs ProcessExpired on a fixed interval until ctx is
// cancelled (same lifecycle pattern as the scheduled-send worker).
func (s *Service) StartReaper(ctx context.Context, interval time.Duration, log *slog.Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for {
					n, err := s.ProcessExpired(ctx, time.Now(), 100)
					if err != nil {
						if ctx.Err() == nil {
							log.Warn("disappear: reaper pass failed", "error", err)
						}
						break
					}
					if n > 0 {
						log.Info("disappear: reaped expired messages", "count", n)
					}
					if n < 100 {
						break
					}
				}
			}
		}
	}()
}
