// Package admin implements the CEO administration surface: user, role and
// credential management plus audit inspection
// (docs/spec/07-business-logic.md "Admin Flow"). Every route is gated
// behind clearance level 1 by the router and every mutation is audited.
package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/auth/password"
	"kisy-backend/internal/users"
)

var (
	ErrNotFound     = errors.New("admin: user not found")
	ErrInvalidRole  = errors.New("admin: role level must be 1..10")
	ErrSelfMutation = errors.New("admin: cannot perform this action on yourself")
	ErrWeakPassword = errors.New("admin: password too weak")
)

// ActorMeta identifies the acting CEO.
type ActorMeta struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

type Service struct {
	pool     *pgxpool.Pool
	users    users.Repository
	sessions auth.SessionRepository
	audit    audit.Recorder
}

func NewService(pool *pgxpool.Pool, usersRepo users.Repository, sessions auth.SessionRepository, rec audit.Recorder) *Service {
	return &Service{pool: pool, users: usersRepo, sessions: sessions, audit: rec}
}

// ListUsers returns a page of accounts (offset pagination).
func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]users.DTO, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	list, err := s.users.List(ctx, s.pool, limit, offset)
	if err != nil {
		return nil, err
	}
	dtos := make([]users.DTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, list[i].ToDTO())
	}
	return dtos, nil
}

// ChangeRole moves a user to a new clearance level. The CEO cannot change
// their own role (guards against self-lockout of the only unrestricted
// account).
func (s *Service) ChangeRole(ctx context.Context, targetID uuid.UUID, newLevel int, actor ActorMeta) error {
	if newLevel < 1 || newLevel > 10 {
		return ErrInvalidRole
	}
	if targetID == actor.UserID {
		return ErrSelfMutation
	}

	current, err := s.users.GetByID(ctx, s.pool, targetID)
	if err != nil {
		return mapNotFound(err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("admin: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.users.UpdateRole(ctx, tx, targetID, newLevel); err != nil {
		return mapNotFound(err)
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     audit.ActionRoleChanged,
		TargetType: "user",
		TargetID:   &targetID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"from": current.RoleID, "to": newLevel},
	}); err != nil {
		return err
	}
	return commit(ctx, tx)
}

// ResetPassword sets a new password chosen by the CEO, forces a change on
// next login and revokes all of the target's sessions.
func (s *Service) ResetPassword(ctx context.Context, targetID uuid.UUID, newPassword string, actor ActorMeta) error {
	if len(newPassword) < 12 || len(newPassword) > 128 {
		return ErrWeakPassword
	}
	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("admin: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.users.AdminResetPasswordHash(ctx, tx, targetID, hash); err != nil {
		return mapNotFound(err)
	}
	if _, err := s.sessions.RevokeAllForUser(ctx, tx, targetID, time.Now().UTC()); err != nil {
		return err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     audit.ActionUserPasswordReset,
		TargetType: "user",
		TargetID:   &targetID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
	}); err != nil {
		return err
	}
	return commit(ctx, tx)
}

// SetActive activates or deactivates an account. Deactivation revokes all
// of the target's sessions. The CEO cannot deactivate themselves.
func (s *Service) SetActive(ctx context.Context, targetID uuid.UUID, active bool, actor ActorMeta) error {
	if targetID == actor.UserID {
		return ErrSelfMutation
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("admin: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.users.SetActive(ctx, tx, targetID, active); err != nil {
		return mapNotFound(err)
	}
	if !active {
		if _, err := s.sessions.RevokeAllForUser(ctx, tx, targetID, time.Now().UTC()); err != nil {
			return err
		}
	}
	action := audit.ActionUserActivated
	if !active {
		action = audit.ActionUserDeactivated
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     action,
		TargetType: "user",
		TargetID:   &targetID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
	}); err != nil {
		return err
	}
	return commit(ctx, tx)
}

func mapNotFound(err error) error {
	if errors.Is(err, users.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func commit(ctx context.Context, tx interface {
	Commit(context.Context) error
}) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("admin: commit: %w", err)
	}
	return nil
}
