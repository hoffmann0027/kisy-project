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
