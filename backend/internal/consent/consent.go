// Package consent records that a person accepted the privacy policy and the
// community rules, and which versions of them.
//
// Google Play requires an app with user-generated content to have rules that
// define objectionable content and behaviour, and to have users accept them
// before they can publish anything. Accepting is therefore not a screen in
// the client but a fact the server keeps: an account cannot be created
// without it, and an existing account is asked again whenever either text
// changes.
//
// The texts themselves live in the frontend (frontend/src/pages/legal), each
// with a version. The versions below must match them — a test in this package
// reads those files and fails if they drift apart.
package consent

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/db"
)

// Current versions of the two documents. Bumping either one asks every
// account to accept again at its next sign-in. Keep them equal to
// PRIVACY_VERSION / RULES_VERSION in frontend/src/pages/legal.
const (
	PrivacyVersion = "2026-10-07"
	RulesVersion   = "2026-10-07"
)

// Acceptance is what a client says the person agreed to.
type Acceptance struct {
	PrivacyVersion string `json:"privacyVersion"`
	RulesVersion   string `json:"rulesVersion"`
}

// Current is the acceptance of the documents in force right now.
func Current() Acceptance {
	return Acceptance{PrivacyVersion: PrivacyVersion, RulesVersion: RulesVersion}
}

// IsCurrent reports whether this acceptance covers the documents in force.
// An acceptance of an older text is not an acceptance of the new one — that
// is the whole point of versioning them.
func (a Acceptance) IsCurrent() bool {
	return a.PrivacyVersion == PrivacyVersion && a.RulesVersion == RulesVersion
}

// ErrNotAccepted: the request did not carry acceptance of the current texts —
// none at all, or a stale bundle showing an older version.
var ErrNotAccepted = errors.New("consent: the current privacy policy and community rules were not accepted")

// Required reports whether an account with these stored versions still has
// to accept: never accepted, or accepted a text that has since changed.
func Required(privacyVersion, rulesVersion *string) bool {
	return privacyVersion == nil || rulesVersion == nil ||
		*privacyVersion != PrivacyVersion || *rulesVersion != RulesVersion
}

// Record stores an acceptance: the current state on the user row, and one
// evidence row per document. Meant to run inside the caller's transaction —
// at sign-up it is part of creating the account, so there is never an
// account without it.
func Record(ctx context.Context, q db.DBTX, userID uuid.UUID, a Acceptance, ipHash string) error {
	if !a.IsCurrent() {
		return ErrNotAccepted
	}
	tag, err := q.Exec(ctx, `
		UPDATE users SET privacy_version = $2, rules_version = $3 WHERE id = $1`,
		userID, a.PrivacyVersion, a.RulesVersion)
	if err != nil {
		return fmt.Errorf("consent: update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("consent: user %s not found", userID)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO legal_acceptances (user_id, document, version, ip_hash)
		VALUES ($1, 'privacy', $2, $4), ($1, 'rules', $3, $4)`,
		userID, a.PrivacyVersion, a.RulesVersion, ipHash); err != nil {
		return fmt.Errorf("consent: record evidence: %w", err)
	}
	return nil
}
