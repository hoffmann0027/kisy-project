// Package invitations implements CEO-issued registration tokens:
// 120-second lifetime, single use, cryptographically random, audited on
// creation and usage (docs/spec/03-backend-architecture.md).
package invitations

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("invitations: not found")

// Invitation mirrors the invitation_tokens table. Only the SHA-256 digest
// of the token is stored; the plaintext exists once, in the create response.
type Invitation struct {
	ID        uuid.UUID
	TokenHash string
	CreatedBy uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    *uuid.UUID
}

// Usable reports whether the invitation can still redeem a registration.
func (i *Invitation) Usable(now time.Time) bool {
	return i.UsedAt == nil && i.ExpiresAt.After(now)
}
