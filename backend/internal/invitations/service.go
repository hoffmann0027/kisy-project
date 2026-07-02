package invitations

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth/token"
)

// CreatorMeta identifies the CEO issuing the invitation, for auditing.
type CreatorMeta struct {
	ActorID   uuid.UUID
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Created is the one-time response payload: the plaintext token is never
// available again after this value is returned.
type Created struct {
	Token     string    `json:"token"`
	CreatorID uuid.UUID `json:"creatorId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type Service struct {
	pool  *pgxpool.Pool
	repo  Repository
	audit audit.Recorder
	ttl   time.Duration
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder, ttl time.Duration) *Service {
	return &Service{pool: pool, repo: repo, audit: rec, ttl: ttl}
}

// Create issues a new invitation token. The caller (handler) has already
// verified the actor is Level 1; timestamps are truncated to microseconds
// so the expires_at = created_at + TTL database check holds exactly after
// PostgreSQL's microsecond rounding.
func (s *Service) Create(ctx context.Context, meta CreatorMeta) (*Created, error) {
	plain, digest, err := token.NewOpaqueToken()
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC().Truncate(time.Microsecond)
	inv := &Invitation{
		TokenHash: digest,
		CreatedBy: meta.ActorID,
		CreatedAt: createdAt,
		ExpiresAt: createdAt.Add(s.ttl),
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("invitations: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Create(ctx, tx, inv); err != nil {
		return nil, err
	}

	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &meta.ActorID,
		Action:     audit.ActionInviteCreated,
		TargetType: "invitation",
		TargetID:   &inv.ID,
		IPHash:     meta.IPHash,
		SessionID:  &meta.SessionID,
		RequestID:  meta.RequestID,
		Metadata:   map[string]any{"expiresAt": inv.ExpiresAt},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("invitations: commit: %w", err)
	}

	return &Created{Token: plain, CreatorID: meta.ActorID, ExpiresAt: inv.ExpiresAt}, nil
}
