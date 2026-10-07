//go:build integration

package moderation_test

import (
	"context"
	"errors"
	"testing"

	"kisy-backend/internal/groups"
	"kisy-backend/internal/moderation"
)

// Audit A-38: a founder deleted their muted community and the sanction
// history went with it (DELETE /groups/{id} → 200, then the admin's
// /communities/{id}/sanctions → 404) — three warnings, then delete and start
// over.

func TestAFounderCannotDeleteAwayLiveSanctions(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	founder := groups.ActorMeta{UserID: e.owner, RoleLevel: 5}

	muted := e.community(t, "Под мутом")
	mute, err := e.mod.Issue(ctx, moderation.IssueInput{GroupID: muted, Kind: moderation.KindMute, Reason: "флуд", Duration: "forever"}, e.ceo)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.groups.Delete(ctx, muted, founder); !errors.Is(err, groups.ErrUnderSanction) {
		t.Fatalf("a muted community was deleted by its founder: %v", err)
	}

	warned := e.community(t, "С предупреждением")
	e.warn(t, warned, "спам")
	if err := e.groups.Delete(ctx, warned, founder); !errors.Is(err, groups.ErrUnderSanction) {
		t.Fatalf("a warned community was deleted by its founder: %v", err)
	}
	if history, err := e.mod.History(ctx, warned); err != nil || len(history) != 1 {
		t.Fatalf("the warning's history: %v %d", err, len(history))
	}

	// Once the sanction is lifted the founder's right is back.
	if _, err := e.mod.Revoke(ctx, mute.Sanction.ID, "", e.ceo); err != nil {
		t.Fatal(err)
	}
	if err := e.groups.Delete(ctx, muted, founder); err != nil {
		t.Fatalf("a founder could not delete after the mute was lifted: %v", err)
	}
	// And the CEO, who issued them, may always delete.
	if err := e.groups.Delete(ctx, warned, groups.ActorMeta{UserID: e.ceo.UserID, RoleLevel: 1}); err != nil {
		t.Fatalf("the CEO could not delete a warned community: %v", err)
	}
}
