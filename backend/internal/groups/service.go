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
	actionGroupCreated      = "group.created"
	actionGroupMemberAdded  = "group.member_added"
	actionGroupDeleted      = "group.deleted"
	actionGroupLevelChanged = "group.level_changed"
)

// ProfileLoader resolves a user's public profile, injected to avoid a
// groups→users import cycle.
type ProfileLoader func(ctx context.Context, userID uuid.UUID) (any, bool)

// ChangePublisher notifies a group's members that the group changed (e.g. its
// avatar) so their clients refetch it. Injected to avoid a groups→ws cycle.
type ChangePublisher func(groupID uuid.UUID)

type Service struct {
	pool     *pgxpool.Pool
	repo     Repository
	audit    audit.Recorder
	profiles ProfileLoader
	changed  ChangePublisher
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder) *Service {
	return &Service{pool: pool, repo: repo, audit: rec}
}

// SetProfileLoader wires member-profile resolution for ListMembers.
func (s *Service) SetProfileLoader(l ProfileLoader) { s.profiles = l }

// SetChangePublisher wires real-time "group changed" notifications.
func (s *Service) SetChangePublisher(p ChangePublisher) { s.changed = p }

// SetAvatar points the group's avatar at an already-stored image URL. Only the
// founder or the CEO may change it; other members get ErrForbidden. Returns
// the refreshed group.
func (s *Service) SetAvatar(ctx context.Context, groupID uuid.UUID, url string, actor ActorMeta) (*Group, error) {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return nil, err
	}
	if g.CreatedBy != actor.UserID && actor.RoleLevel != 1 {
		return nil, ErrForbidden
	}
	if err := s.repo.SetAvatarURL(ctx, s.pool, groupID, url); err != nil {
		return nil, err
	}
	if s.changed != nil {
		s.changed(groupID)
	}
	return s.repo.GetByID(ctx, s.pool, groupID)
}

// SetMinRoleLevel changes a group's minimum clearance (its "level") after
// creation. Only the CEO may do this; other users get ErrForbidden (or
// ErrNotFound if they cannot even see the group, to avoid leaking its
// existence). newLevel must be 1..10 (validated by the handler). Members whose
// clearance no longer satisfies a stricter level simply stop seeing the group;
// they are not force-removed. Returns the refreshed group.
func (s *Service) SetMinRoleLevel(ctx context.Context, groupID uuid.UUID, newLevel int, actor ActorMeta) (*Group, error) {
	g, err := s.repo.GetByID(ctx, s.pool, groupID)
	if err != nil {
		return nil, err // ErrNotFound propagates
	}
	if !access.IsCEO(actor.RoleLevel) {
		if g.IsArchived || !access.CanAccessGroup(actor.RoleLevel, g.MinRoleLevel) {
			return nil, ErrNotFound
		}
		return nil, ErrForbidden
	}
	if newLevel == g.MinRoleLevel {
		return g, nil // no-op; avoid a spurious audit entry and broadcast
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.SetMinRoleLevel(ctx, tx, groupID, newLevel); err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionGroupLevelChanged,
		TargetType: "group",
		TargetID:   &groupID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"from": g.MinRoleLevel, "to": newLevel},
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("groups: commit: %w", err)
	}

	if s.changed != nil {
		s.changed(groupID)
	}
	return s.repo.GetByID(ctx, s.pool, groupID)
}

// ListMembers returns the public profiles of a group's members, if the
// actor may see the group.
func (s *Service) ListMembers(ctx context.Context, groupID uuid.UUID, actor ActorMeta) ([]any, error) {
	if _, err := s.Get(ctx, groupID, actor); err != nil {
		return nil, err
	}
	ids, err := s.repo.ListMemberIDs(ctx, s.pool, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(ids))
	if s.profiles == nil {
		return out, nil
	}
	for _, id := range ids {
		if profile, ok := s.profiles(ctx, id); ok {
			out = append(out, profile)
		}
	}
	return out, nil
}

// CreateInput is validated by the handler before reaching the service.
type CreateInput struct {
	Name         string
	Description  *string
	MinRoleLevel int
}

// Create makes a new group and enrolls the creator as owner. Any user may
// create a group, but its minimum clearance cannot be stronger than the
// creator's own (they must be able to belong to it): min_role_level must
// be >= the creator's role level.
func (s *Service) Create(ctx context.Context, in CreateInput, actor ActorMeta) (*Group, error) {
	if in.MinRoleLevel < actor.RoleLevel {
		return nil, ErrLevelTooHigh
	}

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

// Delete removes a group and everything cascading from it (members, board,
// columns, cards). The CEO may delete any group; otherwise only the
// group's founder may delete it. Group messages are removed too since
// their polymorphic chat_id has no cascading foreign key.
func (s *Service) Delete(ctx context.Context, groupID uuid.UUID, actor ActorMeta) error {
	g, err := s.repo.GetByID(ctx, s.pool, groupID)
	if err != nil {
		return err // ErrNotFound propagates
	}
	// The CEO may delete any group; the founder may delete their own.
	if !access.IsCEO(actor.RoleLevel) && g.CreatedBy != actor.UserID {
		// For anyone else, only reveal a "forbidden" if they can actually
		// see the group; otherwise mask its existence as not-found.
		if g.IsArchived || !access.CanAccessGroup(actor.RoleLevel, g.MinRoleLevel) {
			return ErrNotFound
		}
		return ErrForbidden
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.DeleteGroupMessages(ctx, tx, groupID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, tx, groupID); err != nil {
		return err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionGroupDeleted,
		TargetType: "group",
		TargetID:   &groupID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"name": g.Name, "founder": g.CreatedBy.String()},
	}); err != nil {
		return err
	}

	return commitTx(ctx, tx)
}

func commitTx(ctx context.Context, tx interface {
	Commit(context.Context) error
}) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("groups: commit: %w", err)
	}
	return nil
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

// ClearanceLevel returns a group's audience breadth: its min_role_level, the
// weakest clearance that can access it. Used by message forwarding to compare
// audience sizes (access is verified separately by the caller).
func (s *Service) ClearanceLevel(ctx context.Context, groupID uuid.UUID) (int, error) {
	g, err := s.repo.GetByID(ctx, s.pool, groupID)
	if err != nil {
		return 0, err
	}
	return g.MinRoleLevel, nil
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

// IsFounder reports whether the user created the group.
func (s *Service) IsFounder(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	g, err := s.repo.GetByID(ctx, s.pool, groupID)
	if err != nil {
		return false, err
	}
	return g.CreatedBy == userID, nil
}

// IsMember reports whether the user is a member of the group.
func (s *Service) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	return s.repo.IsMember(ctx, s.pool, groupID, userID)
}
