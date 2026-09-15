// Package users owns the user aggregate: domain model, persistence and
// profile operations.
package users

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/access"
)

var (
	ErrNotFound      = errors.New("users: not found")
	ErrUsernameTaken = errors.New("users: username already taken")
)

// Account kinds. An invited account came through a CEO invitation and has a
// clearance level; a basic one registered without a token and has none — see
// migration 000042 and internal/access.
const (
	KindBasic   = "basic"
	KindInvited = "invited"
)

// User mirrors the users table. RoleID doubles as the clearance level:
// roles.id == roles.level (1 = CEO, 10 = lowest) by schema design.
//
// RoleID is access.NoLevel (0) for a basic account, which is stored as SQL
// NULL. Zero is never a valid level, so it cannot be mistaken for one — but it
// does compare as "better than CEO" under <=, which is why no level check is
// written as a bare comparison; they all go through internal/access.
type User struct {
	ID                  uuid.UUID
	Username            string
	DisplayName         string
	PasswordHash        string
	RoleID              int
	AccountKind         string
	AvatarURL           *string
	Status              string
	LastSeenAt          *time.Time
	IsActive            bool
	FailedLoginAttempts int
	LockedUntil         *time.Time
	MustChangePassword  bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DTO is the public representation from docs/spec/09-api-contracts.md
// (User Object). Sensitive fields never leave the backend.
type DTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	// RoleLevel is null for a basic account. Not 0 and not 10: the client has
	// to be able to tell "outside the hierarchy" from "at the bottom of it",
	// because that is what decides whether levels are shown at all.
	RoleLevel   *int       `json:"roleLevel"`
	AccountKind string     `json:"accountKind"`
	AvatarURL   *string    `json:"avatarUrl"`
	Status      string     `json:"status"`
	IsActive    bool       `json:"isActive"`
	LastSeen    *time.Time `json:"lastSeen"`
	CreatedAt   time.Time  `json:"createdAt"`
	// MustChangePassword is surfaced (omitempty → only when true) so the
	// client can force a password change before granting access. Relevant on
	// the self ("/me", login) response; harmless elsewhere.
	MustChangePassword bool `json:"mustChangePassword,omitempty"`
}

func (u *User) ToDTO() DTO {
	var level *int
	if access.HasLevel(u.RoleID) {
		l := u.RoleID
		level = &l
	}
	return DTO{
		ID:                 u.ID,
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		RoleLevel:          level,
		AccountKind:        u.AccountKind,
		AvatarURL:          u.AvatarURL,
		Status:             u.Status,
		IsActive:           u.IsActive,
		LastSeen:           u.LastSeenAt,
		CreatedAt:          u.CreatedAt,
		MustChangePassword: u.MustChangePassword,
	}
}
