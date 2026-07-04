package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
)

// Service implements profile use-cases (the current user's own account).
type Service struct {
	pool  *pgxpool.Pool
	repo  Repository
	audit audit.Recorder
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder) *Service {
	return &Service{pool: pool, repo: repo, audit: rec}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, s.pool, id)
}

// Directory returns active users the actor may open a chat with (same or
// lower clearance), matching an optional username prefix.
func (s *Service) Directory(ctx context.Context, actorID uuid.UUID, actorLevel int, query string, limit int) ([]DTO, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	list, err := s.repo.Search(ctx, s.pool, actorID, actorLevel, query, limit)
	if err != nil {
		return nil, err
	}
	dtos := make([]DTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, list[i].ToDTO())
	}
	return dtos, nil
}

// PublicProfile returns a user's public DTO, or (zero,false) if the user
// does not exist or is inactive. Used to enrich chat listings with the
// other participant's identity.
func (s *Service) PublicProfile(ctx context.Context, id uuid.UUID) (DTO, bool) {
	u, err := s.repo.GetByID(ctx, s.pool, id)
	if err != nil {
		return DTO{}, false
	}
	return u.ToDTO(), true
}

// TouchLastSeen records the user's last-active time. It is best-effort
// (called from the WebSocket disconnect path) so errors are swallowed.
func (s *Service) TouchLastSeen(ctx context.Context, userID uuid.UUID) {
	_ = s.repo.TouchLastSeen(ctx, s.pool, userID)
}

// ActorMeta carries audit attributes of the acting user.
type ActorMeta struct {
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// ChangeUsername renames the account and audits the change.
func (s *Service) ChangeUsername(ctx context.Context, userID uuid.UUID, newUsername string, meta ActorMeta) (*User, error) {
	current, err := s.repo.GetByID(ctx, s.pool, userID)
	if err != nil {
		return nil, err
	}
	oldUsername := current.Username

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("users: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.UpdateUsername(ctx, tx, userID, newUsername); err != nil {
		return nil, err
	}

	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &userID,
		Action:     audit.ActionUserRenamed,
		TargetType: "user",
		TargetID:   &userID,
		IPHash:     meta.IPHash,
		SessionID:  &meta.SessionID,
		RequestID:  meta.RequestID,
		Metadata:   map[string]any{"from": oldUsername, "to": newUsername},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("users: commit: %w", err)
	}

	return s.repo.GetByID(ctx, s.pool, userID)
}
