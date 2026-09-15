package groups

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/db"
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
	actionGroupCreated         = "group.created"
	actionGroupMemberAdded     = "group.member_added"
	actionGroupDeleted         = "group.deleted"
	actionGroupLevelChanged    = "group.level_changed"
	actionGroupPolicyChanged   = "group.policy_changed"
	actionGroupJoined          = "group.joined"
	actionGroupJoinRequested   = "group.join_requested"
	actionGroupRequestApproved = "group.request_approved"
	actionGroupRequestRejected = "group.request_rejected"
	actionGroupRoleChanged     = "group.role_changed"
)

// ProfileLoader resolves a user's public profile, injected to avoid a
// groups→users import cycle.
type ProfileLoader func(ctx context.Context, userID uuid.UUID) (any, bool)

// ChangePublisher notifies a group's members that the group changed (e.g. its
// avatar) so their clients refetch it. Injected to avoid a groups→ws cycle.
type ChangePublisher func(groupID uuid.UUID)

// DecisionNotifier tells a single user that their join request was decided.
// Injected to avoid a groups→ws cycle; optional.
type DecisionNotifier func(userID, groupID uuid.UUID, approved bool)

type Service struct {
	pool     *pgxpool.Pool
	repo     Repository
	audit    audit.Recorder
	profiles ProfileLoader
	changed  ChangePublisher
	decided  DecisionNotifier
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder) *Service {
	return &Service{pool: pool, repo: repo, audit: rec}
}

// SetProfileLoader wires member-profile resolution for ListMembers.
func (s *Service) SetProfileLoader(l ProfileLoader) { s.profiles = l }

// SetChangePublisher wires real-time "group changed" notifications.
func (s *Service) SetChangePublisher(p ChangePublisher) { s.changed = p }

// SetDecisionNotifier wires per-user join-request decision notifications.
func (s *Service) SetDecisionNotifier(n DecisionNotifier) { s.decided = n }

func (s *Service) broadcast(groupID uuid.UUID) {
	if s.changed != nil {
		s.changed(groupID)
	}
}

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

// ListMembers returns each member's public profile plus their in-group role,
// if the actor may see the group. Shape: [{ "user": <profile>, "role": ... }].
func (s *Service) ListMembers(ctx context.Context, groupID uuid.UUID, actor ActorMeta) ([]any, error) {
	if _, err := s.Get(ctx, groupID, actor); err != nil {
		return nil, err
	}
	members, err := s.repo.ListMembersWithRoles(ctx, s.pool, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(members))
	if s.profiles == nil {
		return out, nil
	}
	for _, m := range members {
		if profile, ok := s.profiles(ctx, m.UserID); ok {
			out = append(out, map[string]any{"user": profile, "role": m.Role})
		}
	}
	return out, nil
}

// ViewerState describes the acting user's standing in a group: whether they
// are a member, their role, and whether they may post under the post policy.
type ViewerState struct {
	Member  bool   `json:"member"`
	Role    string `json:"role"`
	CanPost bool   `json:"canPost"`
	// CanUseWorkspace says whether the board and the calendar are theirs to
	// open. The client hides the tabs by it; the server enforces it regardless.
	CanUseWorkspace bool `json:"canUseWorkspace"`
}

// Viewer returns the actor's own membership/role/post-right for a group. A
// hidden group is masked as not-found.
func (s *Service) Viewer(ctx context.Context, groupID uuid.UUID, actor ActorMeta) (ViewerState, error) {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return ViewerState{}, err
	}
	role, ok, err := s.repo.MemberRole(ctx, s.pool, groupID, actor.UserID)
	if err != nil {
		return ViewerState{}, err
	}
	vs := ViewerState{Member: ok, Role: role}
	vs.CanPost = ok && canPost(g.PostPolicy, role, actor.RoleLevel)
	vs.CanUseWorkspace = ok && canUseWorkspace(g.Kind, role, actor.RoleLevel)
	return vs, nil
}

// EnsureWorkspace returns nil if the actor may use the group's board and
// calendar: a member, and in a community an editor-tier one. A hidden group is
// ErrNotFound, a non-member ErrNotMember, a plain community member ErrForbidden.
func (s *Service) EnsureWorkspace(ctx context.Context, groupID uuid.UUID, actor ActorMeta) error {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return err
	}
	role, ok, err := s.repo.MemberRole(ctx, s.pool, groupID, actor.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotMember
	}
	if !canUseWorkspace(g.Kind, role, actor.RoleLevel) {
		return ErrForbidden
	}
	return nil
}

// HasWorkspace is EnsureWorkspace for someone other than the caller — a card's
// assignee, who must be able to open the board the card is on.
func (s *Service) HasWorkspace(ctx context.Context, groupID, userID uuid.UUID, level int) (bool, error) {
	err := s.EnsureWorkspace(ctx, groupID, ActorMeta{UserID: userID, RoleLevel: level})
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrNotMember), errors.Is(err, ErrForbidden):
		return false, nil
	default:
		return false, err
	}
}

// CreateInput is validated by the handler before reaching the service.
type CreateInput struct {
	Name         string
	Description  *string
	MinRoleLevel int
	// Kind is "group" or "community"; anything else is read as a group.
	Kind string
	// IsPublic puts a community's posts in the shared feed and lets anyone in.
	IsPublic bool
}

// Create makes a new group and enrolls the creator as owner. Any user may
// create a group, but its minimum clearance cannot be stronger than the
// creator's own (they must be able to belong to it): min_role_level must
// be >= the creator's role level.
func (s *Service) Create(ctx context.Context, in CreateInput, actor ActorMeta) (*Group, error) {
	// An account outside the hierarchy has no threshold to offer and is not
	// asked for one: whatever it sends, the group it creates is open to
	// everyone. Asking it to pick a level it does not have would be a form
	// with no valid answer.
	minLevel := in.MinRoleLevel
	if !access.HasLevel(actor.RoleLevel) {
		minLevel = access.NoLevel
	} else if access.HasLevel(minLevel) && minLevel < actor.RoleLevel {
		// A threshold stronger than the creator's own clearance would lock
		// them out of their own group.
		return nil, ErrLevelTooHigh
	}

	kind := in.Kind
	if kind != KindCommunity {
		kind = KindGroup
	}

	g := &Group{
		Name:         in.Name,
		Description:  in.Description,
		MinRoleLevel: minLevel,
		Kind:         kind,
		// Only a community can be public: an ordinary group has no wall to
		// show, so publishing it to the feed would mean nothing.
		IsPublic: kind == KindCommunity && in.IsPublic,
		// A community starts without open discussion, which is the difference
		// between a wall and a group chat. Reusing post_policy rather than
		// adding a second flag: "who may write messages here" is exactly the
		// question, and it is already asked. Its owners can open it later in
		// the group's settings.
		PostPolicy: policyForKind(kind),
		CreatedBy:  actor.UserID,
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

// AddMember puts another user into a group directly. Only the group's owner or
// the CEO may, and the target's clearance must satisfy the group minimum.
//
// Nobody may do it to a community. Joining one is the reader's own decision —
// "Вступить", or a request its editors approve — and a community that could
// enrol people would be one that puts its posts in front of someone who never
// asked for them.
func (s *Service) AddMember(ctx context.Context, groupID, targetID uuid.UUID, targetLevel int, actor ActorMeta) error {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return err
	}
	if g.Kind == KindCommunity {
		return ErrForbidden
	}
	// Until now any user who could merely SEE a group could add anyone to it —
	// themselves included, which walked straight past a request-only join
	// policy. The client only ever offered the button to the founder; the
	// server now agrees with it.
	owner, err := s.isOwnerOrCEO(ctx, g, actor)
	if err != nil {
		return err
	}
	if !owner {
		return ErrForbidden
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

// --- Stage N: access policies, self-join, join requests, roles ---

// record writes one audit event in the given tx.
func (s *Service) record(ctx context.Context, tx db.DBTX, actor ActorMeta, action string, groupID uuid.UUID, meta map[string]any) error {
	gid := groupID
	return s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     action,
		TargetType: "group",
		TargetID:   &gid,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   meta,
	})
}

// isOwnerOrCEO reports whether the actor founded the group, holds the owner
// role, or is the CEO.
func (s *Service) isOwnerOrCEO(ctx context.Context, g *Group, actor ActorMeta) (bool, error) {
	if access.IsCEO(actor.RoleLevel) || g.CreatedBy == actor.UserID {
		return true, nil
	}
	role, ok, err := s.repo.MemberRole(ctx, s.pool, g.ID, actor.UserID)
	if err != nil {
		return false, err
	}
	return ok && role == RoleOwner, nil
}

// canApprove reports whether the actor may approve/reject join requests and
// view the pending list: the CEO, or an editor-tier member.
func (s *Service) canApprove(ctx context.Context, g *Group, actor ActorMeta) (bool, error) {
	if access.IsCEO(actor.RoleLevel) {
		return true, nil
	}
	role, ok, err := s.repo.MemberRole(ctx, s.pool, g.ID, actor.UserID)
	if err != nil {
		return false, err
	}
	return ok && isEditorTier(role), nil
}

// SetPolicies changes a group's join and post policies. Only the founder/owner
// or the CEO may do this. Values are validated. Hidden groups are masked as
// not-found. Returns the refreshed group.
func (s *Service) SetPolicies(ctx context.Context, groupID uuid.UUID, joinPolicy, postPolicy string, actor ActorMeta) (*Group, error) {
	if !validJoinPolicy(joinPolicy) || !validPostPolicy(postPolicy) {
		return nil, ErrBadPolicy
	}
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return nil, err
	}
	ok, err := s.isOwnerOrCEO(ctx, g, actor)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	if g.JoinPolicy == joinPolicy && g.PostPolicy == postPolicy {
		return g, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.SetPolicies(ctx, tx, groupID, joinPolicy, postPolicy); err != nil {
		return nil, err
	}
	if err := s.record(ctx, tx, actor, actionGroupPolicyChanged, groupID, map[string]any{
		"joinPolicy": joinPolicy, "postPolicy": postPolicy,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("groups: commit: %w", err)
	}
	s.broadcast(groupID)
	return s.repo.GetByID(ctx, s.pool, groupID)
}

// Directory returns the groups the actor is cleared to see and is not a member
// of, each with the actor's pending-request status.
func (s *Service) Directory(ctx context.Context, actor ActorMeta) ([]DirectoryEntry, error) {
	return s.repo.ListDirectory(ctx, s.pool, actor.UserID, actor.RoleLevel)
}

// JoinResult reports the outcome of a self-join attempt.
type JoinResult struct {
	Joined bool   // became a member immediately (open policy)
	Status string // "" or "pending" (request policy)
}

// Join lets a cleared user join a group directly (open) or apply (request).
// A user who cannot see the group by clearance gets ErrNotFound.
func (s *Service) Join(ctx context.Context, groupID uuid.UUID, actor ActorMeta) (JoinResult, error) {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return JoinResult{}, err
	}
	// Idempotent: already a member → nothing to do.
	if member, err := s.repo.IsMember(ctx, s.pool, groupID, actor.UserID); err != nil {
		return JoinResult{}, err
	} else if member {
		return JoinResult{Joined: true}, nil
	}

	if g.JoinPolicy == PolicyJoinOpen {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return JoinResult{}, fmt.Errorf("groups: begin tx: %w", err)
		}
		defer tx.Rollback(ctx)
		if err := s.repo.AddMember(ctx, tx, &Member{GroupID: groupID, UserID: actor.UserID, RoleInGroup: RoleMember}); err != nil {
			if errors.Is(err, ErrAlreadyMember) {
				return JoinResult{Joined: true}, nil
			}
			return JoinResult{}, err
		}
		if err := s.record(ctx, tx, actor, actionGroupJoined, groupID, nil); err != nil {
			return JoinResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return JoinResult{}, fmt.Errorf("groups: commit: %w", err)
		}
		s.broadcast(groupID)
		return JoinResult{Joined: true}, nil
	}

	// request policy → create/return a pending request.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return JoinResult{}, fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	jr, err := s.repo.UpsertJoinRequest(ctx, tx, groupID, actor.UserID)
	if err != nil {
		return JoinResult{}, err
	}
	// Only audit a freshly-created request (not a repeat).
	if jr.DecidedAt == nil {
		if err := s.record(ctx, tx, actor, actionGroupJoinRequested, groupID, nil); err != nil {
			return JoinResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return JoinResult{}, fmt.Errorf("groups: commit: %w", err)
	}
	s.broadcast(groupID) // approvers refresh their pending list
	return JoinResult{Status: "pending"}, nil
}

// ListRequests returns the public profiles of users with a pending join
// request. Only editor-tier members or the CEO may view it.
func (s *Service) ListRequests(ctx context.Context, groupID uuid.UUID, actor ActorMeta) ([]any, error) {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return nil, err
	}
	ok, err := s.canApprove(ctx, g, actor)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	ids, err := s.repo.ListPendingRequestUserIDs(ctx, s.pool, groupID)
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

// ApproveRequest admits a pending applicant. Requires editor-tier/CEO. The
// candidate's clearance is re-checked (it may have changed since applying).
func (s *Service) ApproveRequest(ctx context.Context, groupID, targetID uuid.UUID, targetLevel int, actor ActorMeta) error {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return err
	}
	ok, err := s.canApprove(ctx, g, actor)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	if !access.CanAccessGroup(targetLevel, g.MinRoleLevel) {
		// Candidate no longer qualifies; drop the request as rejected.
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("groups: begin tx: %w", err)
		}
		defer tx.Rollback(ctx)
		if err := s.repo.DecidePendingRequest(ctx, tx, groupID, targetID, actor.UserID, "rejected"); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("groups: commit: %w", err)
		}
		return ErrForbidden
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.repo.DecidePendingRequest(ctx, tx, groupID, targetID, actor.UserID, "approved"); err != nil {
		return err // ErrRequestNotFound propagates
	}
	if err := s.repo.AddMember(ctx, tx, &Member{GroupID: groupID, UserID: targetID, RoleInGroup: RoleMember}); err != nil && !errors.Is(err, ErrAlreadyMember) {
		return err
	}
	if err := s.record(ctx, tx, actor, actionGroupRequestApproved, groupID, map[string]any{"userId": targetID.String()}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("groups: commit: %w", err)
	}
	s.broadcast(groupID)
	if s.decided != nil {
		s.decided(targetID, groupID, true)
	}
	return nil
}

// RejectRequest declines a pending applicant. Requires editor-tier/CEO.
func (s *Service) RejectRequest(ctx context.Context, groupID, targetID uuid.UUID, actor ActorMeta) error {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return err
	}
	ok, err := s.canApprove(ctx, g, actor)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.repo.DecidePendingRequest(ctx, tx, groupID, targetID, actor.UserID, "rejected"); err != nil {
		return err
	}
	if err := s.record(ctx, tx, actor, actionGroupRequestRejected, groupID, map[string]any{"userId": targetID.String()}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("groups: commit: %w", err)
	}
	s.broadcast(groupID)
	if s.decided != nil {
		s.decided(targetID, groupID, false)
	}
	return nil
}

// EnsureCanPost returns nil if the actor may POST to the group. Beyond
// membership it enforces the post policy: when editors-only, the actor must be
// editor-tier or the CEO. Read-only members get ErrForbidden. Used by the
// messages module on send.
func (s *Service) EnsureCanPost(ctx context.Context, groupID uuid.UUID, actor ActorMeta) error {
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return err
	}
	role, ok, err := s.repo.MemberRole(ctx, s.pool, groupID, actor.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotMember
	}
	if !canPost(g.PostPolicy, role, actor.RoleLevel) {
		return ErrForbidden
	}
	return nil
}

// SetMemberRole promotes/demotes a member (member ⇄ editor ⇄ moderator). Only
// the founder/owner or CEO may do it; the founder's role cannot be changed and
// only those three assignable roles are allowed.
func (s *Service) SetMemberRole(ctx context.Context, groupID, targetID uuid.UUID, role string, actor ActorMeta) error {
	if role != RoleMember && role != RoleEditor && role != RoleModerator {
		return ErrBadPolicy
	}
	g, err := s.Get(ctx, groupID, actor)
	if err != nil {
		return err
	}
	ok, err := s.isOwnerOrCEO(ctx, g, actor)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	if targetID == g.CreatedBy {
		return ErrForbidden // never demote the founder
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("groups: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.repo.SetMemberRole(ctx, tx, groupID, targetID, role); err != nil {
		return err // ErrNotMember propagates
	}
	if err := s.record(ctx, tx, actor, actionGroupRoleChanged, groupID, map[string]any{"userId": targetID.String(), "role": role}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("groups: commit: %w", err)
	}
	s.broadcast(groupID)
	return nil
}
