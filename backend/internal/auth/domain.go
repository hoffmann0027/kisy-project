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
	// ErrRegistrationClosed: this deployment only accepts invited accounts.
	ErrRegistrationClosed = errors.New("auth: registration without an invitation is closed")
)

// Login/lockout policy per docs/spec/06-security.md ("Account lockout
// after repeated failures"). Failures are counted per name and per source
// address, not per account (audit A-33): an account-wide lock let anyone
// who knew a name — the CEO's, say — keep its owner out for good with five
// wrong passwords every quarter of an hour. Now the guesser locks out only
// themselves, and the owner signs in from their own network as usual.
const (
	MaxLoginAttempts = 5
	LockoutDuration  = 15 * time.Minute
)

// LoginLockedError is a sign-in refused because this name failed too often
// from this address. Unknown names lock exactly like real ones, so the answer
// says nothing about whether an account exists.
type LoginLockedError struct {
	RetryAfter time.Duration
}

func (e *LoginLockedError) Error() string { return ErrAccountLocked.Error() }

func (e *LoginLockedError) Is(target error) bool { return target == ErrAccountLocked }

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
	// MustChangePassword is the owning account's flag, read in the same query
	// as the session so the middleware can enforce it without a second round
	// trip (audit A-16).
	MustChangePassword bool
}

// Active reports whether the session can still be used.
func (s *Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(now)
}

// ClientMeta carries per-request client attributes recorded on sessions
// and audit events.
type ClientMeta struct {
	IPHash string
	// Source is the hashed rate-limit bucket of the client address (an IPv6
	// client is its /64): what failed sign-ins are counted against.
	Source     string
	UserAgent  string
	DeviceName string
	RequestID  string
}
