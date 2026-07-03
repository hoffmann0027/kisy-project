// Package users owns the user aggregate: domain model, persistence and
// profile operations.
package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("users: not found")
	ErrUsernameTaken = errors.New("users: username already taken")
)

// User mirrors the users table. RoleID doubles as the clearance level:
// roles.id == roles.level (1 = CEO, 10 = lowest) by schema design.
type User struct {
	ID                  uuid.UUID
	Username            string
	DisplayName         string
	PasswordHash        string
	RoleID              int
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
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"displayName"`
	RoleLevel   int        `json:"roleLevel"`
	AvatarURL   *string    `json:"avatarUrl"`
	Status      string     `json:"status"`
	IsActive    bool       `json:"isActive"`
	LastSeen    *time.Time `json:"lastSeen"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (u *User) ToDTO() DTO {
	return DTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		RoleLevel:   u.RoleID,
		AvatarURL:   u.AvatarURL,
		Status:      u.Status,
		IsActive:    u.IsActive,
		LastSeen:    u.LastSeenAt,
		CreatedAt:   u.CreatedAt,
	}
}
