package token

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/users"
)

const testSecret = "test-secret-at-least-32-characters-long"

func TestIssueAndParseAccess(t *testing.T) {
	m := NewManager(testSecret, 15*time.Minute)
	userID := uuid.New()
	sessionID := uuid.New()

	raw, expiresAt, err := m.IssueAccess(userID, sessionID, 3, users.KindInvited)
	if err != nil {
		t.Fatalf("IssueAccess() error: %v", err)
	}
	if time.Until(expiresAt) < 14*time.Minute {
		t.Fatalf("expiry too soon: %v", expiresAt)
	}

	claims, err := m.ParseAccess(raw)
	if err != nil {
		t.Fatalf("ParseAccess() error: %v", err)
	}
	if claims.UserID != userID || claims.SessionID != sessionID || claims.RoleLevel != 3 {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

func TestParseAccessExpired(t *testing.T) {
	m := NewManager(testSecret, -1*time.Minute)
	raw, _, err := m.IssueAccess(uuid.New(), uuid.New(), 1, users.KindInvited)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.ParseAccess(raw); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestParseAccessWrongSecret(t *testing.T) {
	issuerM := NewManager(testSecret, time.Minute)
	raw, _, err := issuerM.IssueAccess(uuid.New(), uuid.New(), 1, users.KindInvited)
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewManager("another-secret-also-32-characters-xx", time.Minute)
	if _, err := verifier.ParseAccess(raw); err != ErrInvalid {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestParseAccessGarbage(t *testing.T) {
	m := NewManager(testSecret, time.Minute)
	for _, raw := range []string{"", "abc", "a.b.c"} {
		if _, err := m.ParseAccess(raw); err != ErrInvalid {
			t.Errorf("ParseAccess(%q): expected ErrInvalid, got %v", raw, err)
		}
	}
}

func TestNewOpaqueToken(t *testing.T) {
	plain, digest, err := NewOpaqueToken()
	if err != nil {
		t.Fatalf("NewOpaqueToken() error: %v", err)
	}
	if len(plain) < 40 {
		t.Fatalf("opaque token too short: %d chars", len(plain))
	}
	if HashOpaqueToken(plain) != digest {
		t.Fatal("digest does not match HashOpaqueToken(plain)")
	}

	plain2, digest2, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if plain == plain2 || digest == digest2 {
		t.Fatal("two opaque tokens are identical; randomness broken")
	}
}

// An account outside the role hierarchy has no level, so its token carries
// zero — a value ParseAccess rejected outright before. The relaxation must be
// exactly as wide as that one case and no wider: zero is acceptable only when
// the token also says, in a signed claim, that this account has no level.

func TestBasicAccountTokenRoundTrips(t *testing.T) {
	m := NewManager(testSecret, time.Minute)
	raw, _, err := m.IssueAccess(uuid.New(), uuid.New(), 0, users.KindBasic)
	if err != nil {
		t.Fatalf("IssueAccess() error: %v", err)
	}
	claims, err := m.ParseAccess(raw)
	if err != nil {
		t.Fatalf("ParseAccess() error: %v", err)
	}
	if claims.RoleLevel != 0 || claims.Kind != users.KindBasic {
		t.Fatalf("got level=%d kind=%q, want 0/basic", claims.RoleLevel, claims.Kind)
	}
}

func TestLevelZeroWithoutTheBasicClaimIsRejected(t *testing.T) {
	// A token whose "lvl" was stripped must not be readable as a level-less
	// account: zero compares as stronger than the CEO wherever a level is
	// compared, so this is the difference between a guard and a hole.
	m := NewManager(testSecret, time.Minute)
	raw, _, err := m.IssueAccess(uuid.New(), uuid.New(), 0, "")
	if err != nil {
		t.Fatalf("IssueAccess() error: %v", err)
	}
	if _, err := m.ParseAccess(raw); err == nil {
		t.Fatal("a token claiming no level and no kind must be rejected")
	}
}

func TestBasicClaimCannotCarryALevel(t *testing.T) {
	m := NewManager(testSecret, time.Minute)
	raw, _, err := m.IssueAccess(uuid.New(), uuid.New(), 1, users.KindBasic)
	if err != nil {
		t.Fatalf("IssueAccess() error: %v", err)
	}
	if _, err := m.ParseAccess(raw); err == nil {
		t.Fatal("a basic account holding CEO clearance must be rejected")
	}
}

func TestTokenIssuedBeforeAccountKindsStillWorks(t *testing.T) {
	// Sessions in flight during the deploy have no "knd" claim. They belong to
	// invited accounts — those were the only kind there was.
	m := NewManager(testSecret, time.Minute)
	raw, _, err := m.IssueAccess(uuid.New(), uuid.New(), 4, "")
	if err != nil {
		t.Fatalf("IssueAccess() error: %v", err)
	}
	claims, err := m.ParseAccess(raw)
	if err != nil {
		t.Fatalf("ParseAccess() error: %v", err)
	}
	if claims.Kind != users.KindInvited {
		t.Fatalf("kind = %q, want invited", claims.Kind)
	}
}
