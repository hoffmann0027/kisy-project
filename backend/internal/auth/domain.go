// Package auth implements authentication and session lifecycle: login,
// registration by invitation, refresh-token rotation, logout and the
// RBAC middleware.
package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrAccountLocked      = errors.New("auth: account temporarily locked")
	ErrInvalidRefresh     = errors.New("auth: invalid refresh token")
	ErrInvalidInvite      = errors.New("auth: invalid or expired invitation token")
)

// Login/lockout policy per docs/spec/06-security.md ("Account lockout
// after repeated failures").
const (
	MaxLoginAttempts = 5
	LockoutDuration  = 15 * time.Minute
)

// DefaultRegisteredRoleLevel is the clearance assigned to accounts created
// through an invitation. The spec does not attach a role to invitations,
// so new users start at the lowest clearance and are promoted by the CEO.
const DefaultRegisteredRoleLevel = 10

// Session mirrors the sessions table: one row per authenticated device,
// keyed by the hash of its current refresh token.
type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string
	DeviceName       *string
	UserAgent        *string
	IPHash           string
	CreatedAt        time.Time
	LastUsedAt       time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
}

// Active reports whether the session can still be used.
func (s *Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(now)
}

// ClientMeta carries per-request client attributes recorded on sessions
// and audit events.
type ClientMeta struct {
	IPHash     string
	UserAgent  string
	DeviceName string
	RequestID  string
}
