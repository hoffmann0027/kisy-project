// Package users owns the user aggregate: domain model, persistence and
// profile operations.
package users

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/access"
	"kisy-backend/internal/consent"
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
	ID                 uuid.UUID
	Username           string
	DisplayName        string
	PasswordHash       string
	RoleID             int
	AccountKind        string
	AvatarURL          *string
	Status             string
	LastSeenAt         *time.Time
	IsActive           bool
	MustChangePassword bool
	// DisplayNameNeedsChange is set by migration 46 on accounts whose existing
	// name broke the display-name rule or collided with an earlier account's.
	// Cleared by choosing a new name.
	DisplayNameNeedsChange bool
	// VerifiedAt is when the CEO gave this account the verification mark;
	// nil when it has none (migration 47).
	VerifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// PrivacyVersion / RulesVersion are the versions of the privacy policy and
	// the community rules this account last accepted; nil when it never did
	// (migration 55, internal/consent).
	PrivacyVersion *string
	RulesVersion   *string
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
	// VerifiedAt is non-null while the account carries the verification mark.
	VerifiedAt *time.Time `json:"verifiedAt"`
	// MustChangePassword is surfaced (omitempty → only when true) so the
	// client can force a password change before granting access. Relevant on
	// the self ("/me", login) response; harmless elsewhere.
	MustChangePassword bool `json:"mustChangePassword,omitempty"`
	// DisplayNameNeedsChange (omitempty → only when true) makes the client
	// stop at a blocking "choose a new name" screen, like MustChangePassword.
	DisplayNameNeedsChange bool `json:"displayNameNeedsChange,omitempty"`
	// ConsentRequired (omitempty → only when true) stops the client at the
	// screen with the privacy policy and community rules: never accepted, or
	// accepted a text that has changed since. Only ever set by ToSelfDTO —
	// whether someone else has accepted the rules is nobody's business.
	ConsentRequired bool `json:"consentRequired,omitempty"`
}

func (u *User) ToDTO() DTO {
	var level *int
	if access.HasLevel(u.RoleID) {
		l := u.RoleID
		level = &l
	}
	return DTO{
		ID:                     u.ID,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		RoleLevel:              level,
		AccountKind:            u.AccountKind,
		AvatarURL:              u.AvatarURL,
		Status:                 u.Status,
		IsActive:               u.IsActive,
		LastSeen:               u.LastSeenAt,
		CreatedAt:              u.CreatedAt,
		VerifiedAt:             u.VerifiedAt,
		MustChangePassword:     u.MustChangePassword,
		DisplayNameNeedsChange: u.DisplayNameNeedsChange,
	}
}

// ToSelfDTO is the DTO of the account making the request: ToDTO plus what only
// its owner should learn about it — whether it still has to accept the
// current privacy policy and community rules.
func (u *User) ToSelfDTO() DTO {
	d := u.ToDTO()
	d.ConsentRequired = consent.Required(u.PrivacyVersion, u.RulesVersion)
	return d
}
