// Package groups owns group chats and their clearance-gated visibility.
package groups

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/access"
)

var (
	ErrNotFound      = errors.New("groups: not found")
	ErrNotMember     = errors.New("groups: actor is not a member")
	ErrAlreadyMember = errors.New("groups: user already a member")
	ErrForbidden     = errors.New("groups: not permitted")
	// ErrLevelTooHigh is returned when a user tries to create a group whose
	// minimum clearance is stronger than their own (lower level number).
	ErrLevelTooHigh = errors.New("groups: cannot create a group above your clearance")
	// ErrRequestNotFound is returned when no pending join request exists.
	ErrRequestNotFound = errors.New("groups: join request not found")
	// ErrBadPolicy is returned for an unknown join/post policy value.
	ErrBadPolicy = errors.New("groups: invalid policy value")
)

// Join and post policy values (Stage N — group access settings).
const (
	PolicyJoinOpen    = "open"    // any cleared user self-joins
	PolicyJoinRequest = "request" // cleared user applies; an editor approves
	PolicyPostAll     = "all"     // any member may post
	PolicyPostEditors = "editors" // only owner/editor/moderator (or CEO) may post
)

// In-group roles (role_in_group). The "editor tier" — owner, editor,
// moderator (and, cross-cutting, the CEO by clearance) — may post in an
// editors-only group and approve join requests.
const (
	RoleMember    = "member"
	RoleModerator = "moderator"
	RoleEditor    = "editor"
	RoleOwner     = "owner"
)

// isEditorTier reports whether an in-group role belongs to the editor tier
// (may post when post_policy=editors and may approve join requests).
func isEditorTier(role string) bool {
	switch role {
	case RoleOwner, RoleEditor, RoleModerator:
		return true
	default:
		return false
	}
}

// Group mirrors the groups table. MinRoleLevel is the weakest clearance
// (largest level number) allowed to see the group.
type Group struct {
	ID           uuid.UUID
	Name         string
	Description  *string
	AvatarURL    *string
	MinRoleLevel int
	JoinPolicy   string
	PostPolicy   string
	CreatedBy    uuid.UUID
	IsArchived   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Member struct {
	GroupID     uuid.UUID
	UserID      uuid.UUID
	RoleInGroup string
	MutedUntil  *time.Time
	JoinedAt    time.Time
}

// JoinRequest mirrors a row of group_join_requests.
type JoinRequest struct {
	ID          uuid.UUID
	GroupID     uuid.UUID
	UserID      uuid.UUID
	Status      string
	RequestedAt time.Time
	DecidedBy   *uuid.UUID
	DecidedAt   *time.Time
}

// DTO is the API representation of a group.
type DTO struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	AvatarURL    *string   `json:"avatarUrl"`
	MinRoleLevel int       `json:"minRoleLevel"`
	JoinPolicy   string    `json:"joinPolicy"`
	PostPolicy   string    `json:"postPolicy"`
	CreatedBy    uuid.UUID `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (g *Group) ToDTO() DTO {
	return DTO{
		ID:           g.ID,
		Name:         g.Name,
		Description:  g.Description,
		AvatarURL:    g.AvatarURL,
		MinRoleLevel: g.MinRoleLevel,
		JoinPolicy:   g.JoinPolicy,
		PostPolicy:   g.PostPolicy,
		CreatedBy:    g.CreatedBy,
		CreatedAt:    g.CreatedAt,
	}
}

// DirectoryDTO is a group in the "find a group" catalogue: the group plus the
// actor's own pending-request status (empty when none).
type DirectoryDTO struct {
	DTO
	RequestStatus string `json:"requestStatus,omitempty"` // "" | "pending"
	// E2EEJoinNeedsApproval flags that, even for an open group, joining is
	// mediated by an editor because MLS add must be performed by a member.
	// (Reserved; plaintext groups self-join instantly.)
}

// canPost reports whether an actor with the given in-group role and clearance
// level may post to a group under the given post policy. The CEO always may.
func canPost(postPolicy, role string, level int) bool {
	if postPolicy != PolicyPostEditors {
		return true // any member may post
	}
	return access.IsCEO(level) || isEditorTier(role)
}

// validJoinPolicy / validPostPolicy guard incoming values.
func validJoinPolicy(p string) bool { return p == PolicyJoinOpen || p == PolicyJoinRequest }
func validPostPolicy(p string) bool { return p == PolicyPostAll || p == PolicyPostEditors }
