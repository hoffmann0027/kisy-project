package moderation

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"kisy-backend/internal/groups"
)

// GroupRoles answers "does this actor run the group?" with the groups
// service's own visibility and roles: the founder, or an owner, editor or
// moderator. A group the actor cannot see is ErrNotFound, not a refusal.
type GroupRoles struct {
	Groups *groups.Service
}

func (r GroupRoles) EditorTier(ctx context.Context, groupID uuid.UUID, actor ActorMeta) (bool, error) {
	g := groups.ActorMeta{UserID: actor.UserID, RoleLevel: actor.RoleLevel, SessionID: actor.SessionID}
	group, err := r.Groups.Get(ctx, groupID, g)
	if errors.Is(err, groups.ErrNotFound) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if group.CreatedBy == actor.UserID {
		return true, nil
	}
	vs, err := r.Groups.Viewer(ctx, groupID, g)
	if err != nil {
		return false, err
	}
	switch vs.Role {
	case groups.RoleOwner, groups.RoleEditor, groups.RoleModerator:
		return true, nil
	}
	return false, nil
}
