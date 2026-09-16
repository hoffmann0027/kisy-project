// Package moderation is the CEO's hand on groups and communities: warnings,
// mutes and deletion (migration 48, docs/spec/07-business-logic.md).
//
// The rules, all enforced here:
//   - A warning counts while it is not revoked. The third active warning
//     deletes the group, exactly as a deletion issued by hand would.
//   - A mute lasts a day, a week, a month or indefinitely. While it is live
//     the community's posts are left out of GET /feed — in SQL, in
//     internal/posts — but its editors may keep publishing to its own wall.
//   - Deletion is soft. For RestoreWindow the CEO can bring the group back with
//     everything in it; the active warnings are then cut to WarnLimit-1, so the
//     next warning deletes it again. After the window the daily purge removes it.
//   - Every action needs a reason, which its recipients are shown, and every
//     action is in the audit log.
package moderation

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sanction kinds, mirroring the group_sanctions CHECK constraint.
const (
	KindWarn   = "warn"
	KindMute   = "mute"
	KindDelete = "delete"
)

const (
	// WarnLimit active warnings delete a group.
	WarnLimit = 3
	// RestoreWindow is how long a deleted group can still be restored.
	RestoreWindow = 30 * 24 * time.Hour
	// PurgeInterval is how often groups past their window are removed for good.
	PurgeInterval = 24 * time.Hour
	// maxReasonLength mirrors the CHECK constraint.
	maxReasonLength = 1000
)

// MuteDurations are the choices the CEO has; "forever" is an indefinite mute.
var MuteDurations = map[string]time.Duration{
	"1d":      24 * time.Hour,
	"7d":      7 * 24 * time.Hour,
	"30d":     30 * 24 * time.Hour,
	"forever": 0,
}

var (
	ErrNotFound        = errors.New("moderation: not found")
	ErrForbidden       = errors.New("moderation: not permitted")
	ErrInvalidKind     = errors.New("moderation: unknown sanction kind")
	ErrReasonRequired  = errors.New("moderation: a reason is required")
	ErrReasonTooLong   = errors.New("moderation: reason too long")
	ErrInvalidDuration = errors.New("moderation: unknown mute duration")
	// ErrGroupDeleted: the group is already deleted; restore it first.
	ErrGroupDeleted = errors.New("moderation: group is deleted")
	// ErrNotDeleted: restore was asked for a group that is not deleted.
	ErrNotDeleted = errors.New("moderation: group is not deleted")
	// ErrRestoreExpired: the restore window has closed.
	ErrRestoreExpired = errors.New("moderation: restore window has passed")
	// ErrNotRevocable: only a live warning or mute can be revoked.
	ErrNotRevocable = errors.New("moderation: sanction cannot be revoked")
)

// ActorMeta identifies the acting user.
type ActorMeta struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	RoleLevel int
	IPHash    string
	RequestID string
}

// Sanction mirrors a group_sanctions row.
type Sanction struct {
	ID         uuid.UUID  `json:"id"`
	GroupID    uuid.UUID  `json:"groupId"`
	Kind       string     `json:"kind"`
	Reason     string     `json:"reason"`
	IssuedBy   uuid.UUID  `json:"issuedBy"`
	IssuedAt   time.Time  `json:"issuedAt"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
	RevokedBy  *uuid.UUID `json:"revokedBy"`
	RevokeNote *string    `json:"revokeNote"`
}

// Active reports whether the sanction still has an effect at now.
func (s *Sanction) Active(now time.Time) bool {
	if s.RevokedAt != nil {
		return false
	}
	if s.Kind == KindMute && s.ExpiresAt != nil && !s.ExpiresAt.After(now) {
		return false
	}
	return true
}

// groupRow is the part of a group moderation reads.
type groupRow struct {
	ID        uuid.UUID
	Name      string
	Kind      string
	CreatedBy uuid.UUID
	DeletedAt *time.Time
}

// GroupSummary is one row of the CEO's communities list.
type GroupSummary struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	AvatarURL   *string    `json:"avatarUrl"`
	IsPublic    bool       `json:"isPublic"`
	VerifiedAt  *time.Time `json:"verifiedAt"`
	MemberCount int        `json:"memberCount"`
	ActiveWarns int        `json:"activeWarns"`
	WarnLimit   int        `json:"warnLimit"`
	// Muted is true while a mute is live; MutedUntil is nil for an indefinite one.
	Muted      bool       `json:"muted"`
	MutedUntil *time.Time `json:"mutedUntil"`
}

// DeletedGroup is one row of the CEO's "deleted" list.
type DeletedGroup struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	AvatarURL    *string   `json:"avatarUrl"`
	DeletedAt    time.Time `json:"deletedAt"`
	PurgeAt      time.Time `json:"purgeAt"`
	DeleteReason string    `json:"deleteReason"`
}

// ActiveSanctions is what a group's own editors see in the banner.
type ActiveSanctions struct {
	Warns     []Sanction `json:"warns"`
	WarnLimit int        `json:"warnLimit"`
	Mute      *Sanction  `json:"mute"`
}

// Outcome is the result of issuing a sanction.
type Outcome struct {
	Sanction    Sanction `json:"sanction"`
	ActiveWarns int      `json:"activeWarns"`
	// Deleted is true when this action deleted the group — a deletion, or the
	// warning that reached WarnLimit.
	Deleted bool `json:"deleted"`
}
