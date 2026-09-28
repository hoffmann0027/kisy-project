package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// TestResetPasswordUsesTheSharedPolicy pins the admin half of audit D-14: the
// CEO's reset checked the length only, so it could hand out a password weaker
// than any user was allowed to choose — twelve letters with no digit.
//
// Only refusals are exercised: the policy is checked before the transaction
// opens, so a valid password would reach the (absent) pool.
func TestResetPasswordUsesTheSharedPolicy(t *testing.T) {
	svc := &Service{}
	target := uuid.New()

	for _, weak := range []string{
		"aaaaaaaaaaaa",   // twelve letters, no digit
		"123456789012",   // digits only
		"парольбезцифры", // Cyrillic, no digit
		"short1",         // too short
	} {
		err := svc.ResetPassword(context.Background(), target, weak, ActorMeta{})
		if !errors.Is(err, ErrWeakPassword) {
			t.Fatalf("ResetPassword(%q) = %v, want ErrWeakPassword", weak, err)
		}
	}
}
