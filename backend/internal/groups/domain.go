// Package groups owns group chats and their clearance-gated visibility.
package groups

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("groups: not found")
	ErrNotMember     = errors.New("groups: actor is not a member")
	ErrAlreadyMember = errors.New("groups: user already a member")
)

// Group mirrors the groups table. MinRoleLevel is the weakest clearance
// (largest level number) allowed to see the group.
type Group struct {
	ID           uuid.UUID
	Name         string
	Description  *string
	AvatarURL    *string
	MinRoleLevel int
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

// DTO is the API representation of a group.
type DTO struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	AvatarURL    *string   `json:"avatarUrl"`
	MinRoleLevel int       `json:"minRoleLevel"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (g *Group) ToDTO() DTO {
	return DTO{
		ID:           g.ID,
		Name:         g.Name,
		Description:  g.Description,
		AvatarURL:    g.AvatarURL,
		MinRoleLevel: g.MinRoleLevel,
		CreatedAt:    g.CreatedAt,
	}
}
