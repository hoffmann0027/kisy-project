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
	"kisy-backend/internal/groups"
	"kisy-backend/internal/users"
)

var (
	ErrNotFound    = errors.New("admin: user not found")
	ErrInvalidRole = errors.New("admin: role level must be 1..10")
	// ErrNotInvited: the account registered without an invitation and stands
	// outside the role hierarchy, so it cannot be given a level.
	ErrNotInvited   = errors.New("admin: account registered without an invitation has no place in the role hierarchy")
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

	// groups backs the verification mark on groups/communities; optional so
	// the user-only admin surface keeps working without it.
	groups       groups.Repository
	userChanged  func(ctx context.Context, id uuid.UUID)
	groupChanged func(id uuid.UUID)

	// kick ends the target's open sockets once their sessions are revoked
	// (audit A-03); nil leaves that to the sockets' own re-check.
	kick auth.SessionKicker
}

// SetSessionKicker wires the connection kick for role changes, password
// resets and deactivations.
func (s *Service) SetSessionKicker(k auth.SessionKicker) { s.kick = k }

// commitAndKick commits a transaction that revoked every session of target
// and only then ends the target's sockets, so a reconnect cannot slip in
// before the revocation is visible.
func (s *Service) commitAndKick(ctx context.Context, tx interface{ Commit(context.Context) error }, target uuid.UUID) error {
	if err := commit(ctx, tx); err != nil {
		return err
	}
	if s.kick != nil {
		s.kick.KickUser(target, uuid.Nil)
	}
	return nil
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
	// A basic account is not at the bottom of the hierarchy, it is outside it.
	// Granting it a level would make it an invited account without anyone
	// having invited it — the one thing the invitation model exists to
	// prevent — and the database refuses the combination anyway.
	if current.AccountKind != users.KindInvited {
		return ErrNotInvited
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("admin: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.users.UpdateRole(ctx, tx, targetID, newLevel); err != nil {
		return mapNotFound(err)
	}
	// The access token carries the clearance in its `lvl` claim and
	// RequireClearance trusts that claim, so a demotion would otherwise only
	// take effect when the token expires — up to JWT_ACCESS_TTL of continued
	// access to /admin, /invites and the old clearance band's groups. Cutting
	// the sessions forces a re-login, which re-issues the claim at the new
	// level; that re-login is the intended cost of changing someone's clearance.
	if _, err := s.sessions.RevokeAllForUser(ctx, tx, targetID, time.Now().UTC()); err != nil {
		return err
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
	return s.commitAndKick(ctx, tx, targetID)
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
	return s.commitAndKick(ctx, tx, targetID)
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
	if !active {
		return s.commitAndKick(ctx, tx, targetID)
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
