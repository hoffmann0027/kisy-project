package groups

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
)

// ActorMeta identifies the acting user for permission checks and auditing.
type ActorMeta struct {
	UserID    uuid.UUID
	RoleLevel int
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Action names for the audit log.
const (
	actionGroupCreated     = "group.created"
	actionGroupMemberAdded = "group.member_added"
)

type Service struct {
	pool  *pgxpool.Pool
	repo  Repository
	audit audit.Recorder
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder) *Service {
	return &Service{pool: pool, repo: repo, audit: rec}
}

// CreateInput is validated by the handler before reaching the service.
type CreateInput struct {
	Name         string
	Description  *string
	MinRoleLevel int
}

// Create makes a new group and enrolls the creator as owner. The caller
// (handler) has already verified the actor is the CEO.
func (s *Service) Create(ctx context.Context, in CreateInput, actor ActorMeta) (*Group, error) {
	g := &Group{
		Name:         in.Name,
		Description:  in.Description,
		MinRoleLevel: in.MinRoleLevel,
		CreatedBy:    actor.UserID,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Create(ctx, tx, g); err != nil {
		return nil, err
	}
	if err := s.repo.AddMember(ctx, tx, &Member{GroupID: g.ID, UserID: actor.UserID, RoleInGroup: "owner"}); err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionGroupCreated,
		TargetType: "group",
		TargetID:   &g.ID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"minRoleLevel": g.MinRoleLevel},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("groups: commit: %w", err)
	}
	return g, nil
}

// ListVisible returns the groups the actor is cleared to see.
func (s *Service) ListVisible(ctx context.Context, actor ActorMeta) ([]Group, error) {
	return s.repo.ListVisible(ctx, s.pool, actor.RoleLevel)
}

// Get returns a group only if the actor is cleared to see it; otherwise it
// returns ErrNotFound so callers cannot distinguish "hidden" from
// "missing" (information-leak prevention, docs/spec/01 §5).
func (s *Service) Get(ctx context.Context, id uuid.UUID, actor ActorMeta) (*Group, error) {
	g, err := s.repo.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if g.IsArchived || !access.CanAccessGroup(actor.RoleLevel, g.MinRoleLevel) {
		return nil, ErrNotFound
	}
	return g, nil
}

// AddMember adds a user to a group. The actor must be able to see the
// group, and the target's clearance must also satisfy the group minimum.
func (s *Service) AddMember(ctx context.Context, groupID, targetID uuid.UUID, targetLevel int, actor ActorMeta) error {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return err
	}
	if !access.CanAccessGroup(targetLevel, g.MinRoleLevel) {
		return ErrNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.AddMember(ctx, tx, &Member{GroupID: groupID, UserID: targetID, RoleInGroup: "member"}); err != nil {
		return err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionGroupMemberAdded,
		TargetType: "group",
		TargetID:   &groupID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"memberId": targetID.String()},
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("groups: commit: %w", err)
	}
	return nil
}

// EnsureMember returns nil if the actor may post to the group. Used by the
// messages module. A hidden group is reported as ErrNotFound.
func (s *Service) EnsureMember(ctx context.Context, groupID uuid.UUID, actor ActorMeta) error {
	if _, err := s.Get(ctx, groupID, actor); err != nil {
		return err
	}
	member, err := s.repo.IsMember(ctx, s.pool, groupID, actor.UserID)
	if err != nil {
		return err
	}
	if !member {
		return ErrNotMember
	}
	return nil
}

// MemberIDs returns the user IDs of every member, for real-time event
// fan-out.
func (s *Service) MemberIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.ListMemberIDs(ctx, s.pool, groupID)
}
