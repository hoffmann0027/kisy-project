package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/users"
)

// The verification mark (migration 47): given and taken away by the CEO, for
// accounts and for groups/communities, every change in the audit log.

// ErrGroupNotFound is returned when the group to verify does not exist.
var ErrGroupNotFound = errors.New("admin: group not found")

// verificationSearchLimit bounds one search: the screen is a lookup, not a list.
const verificationSearchLimit = 30

// SetGroupsRepository wires the groups side of verification (and, later, the
// community moderation screens).
func (s *Service) SetGroupsRepository(r groups.Repository) { s.groups = r }

// SetUserChanged / SetGroupChanged tell clients to refetch a profile or a
// group after its mark changed, so the badge appears without a reload.
func (s *Service) SetUserChanged(fn func(ctx context.Context, id uuid.UUID)) { s.userChanged = fn }
func (s *Service) SetGroupChanged(fn func(id uuid.UUID))                     { s.groupChanged = fn }

// VerificationSearch is what the CEO's verification tab shows for a query.
type VerificationSearch struct {
	Users  []users.DTO  `json:"users"`
	Groups []groups.DTO `json:"groups"`
}

// SearchForVerification finds accounts and groups by name for the mark.
func (s *Service) SearchForVerification(ctx context.Context, query string) (VerificationSearch, error) {
	out := VerificationSearch{Users: []users.DTO{}, Groups: []groups.DTO{}}
	us, err := s.users.AdminSearch(ctx, s.pool, query, verificationSearchLimit)
	if err != nil {
		return out, err
	}
	for i := range us {
		out.Users = append(out.Users, us[i].ToDTO())
	}
	if s.groups != nil {
		gs, err := s.groups.AdminSearch(ctx, s.pool, query, verificationSearchLimit)
		if err != nil {
			return out, err
		}
		for i := range gs {
			out.Groups = append(out.Groups, gs[i].ToDTO())
		}
	}
	return out, nil
}

// SetUserVerified gives or takes away an account's mark.
func (s *Service) SetUserVerified(ctx context.Context, targetID uuid.UUID, verified bool, actor ActorMeta) (*users.DTO, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("admin: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.users.SetVerified(ctx, tx, targetID, actor.UserID, verified); err != nil {
		if errors.Is(err, users.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	action := audit.ActionUserUnverified
	if verified {
		action = audit.ActionUserVerified
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
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("admin: commit: %w", err)
	}

	if s.userChanged != nil {
		s.userChanged(ctx, targetID)
	}
	u, err := s.users.GetByID(ctx, s.pool, targetID)
	if err != nil {
		return nil, err
	}
	dto := u.ToDTO()
	return &dto, nil
}

// SetGroupVerified gives or takes away a group's or community's mark.
func (s *Service) SetGroupVerified(ctx context.Context, groupID uuid.UUID, verified bool, actor ActorMeta) (*groups.DTO, error) {
	if s.groups == nil {
		return nil, ErrGroupNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("admin: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.groups.SetVerified(ctx, tx, groupID, actor.UserID, verified); err != nil {
		if errors.Is(err, groups.ErrNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	action := audit.ActionGroupUnverified
	if verified {
		action = audit.ActionGroupVerified
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     action,
		TargetType: "group",
		TargetID:   &groupID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("admin: commit: %w", err)
	}

	if s.groupChanged != nil {
		s.groupChanged(groupID)
	}
	g, err := s.groups.GetByID(ctx, s.pool, groupID)
	if err != nil {
		return nil, err
	}
	dto := g.ToDTO()
	return &dto, nil
}
