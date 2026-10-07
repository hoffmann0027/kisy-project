//go:build integration

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/feedback"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

// Audit A-19: an account outside the hierarchy has no staff directory by
// design, yet three paths handed it one — the feedback board (every author's
// login and level), the member list of any group it could see (closed ones
// and every public community), and presence for any user id it named.

func TestFeedbackDoesNotMapTheCompanyForABasicAccount(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	manager := testdb.SeedUser(t, pool, "manager_"+uuid.NewString()[:6], 2)
	svc := feedback.NewService(pool, feedback.NewPostgresRepository())
	if _, err := svc.Create(ctx, manager, "предложение"); err != nil {
		t.Fatal(err)
	}

	basic := feedback.Actor{UserID: uuid.New(), RoleLevel: 0}
	page, err := svc.List(ctx, basic, "", 10)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list: %v %d", err, len(page.Items))
	}
	author := page.Items[0].Author
	if author.Username != "" || author.RoleLevel != nil {
		t.Fatalf("a basic account saw an author's login %q and level %v", author.Username, author.RoleLevel)
	}
	if author.DisplayName == "" {
		t.Fatal("the display name is public and should stay")
	}

	// Someone in the hierarchy still sees who is who.
	invited := feedback.Actor{UserID: uuid.New(), RoleLevel: 5}
	page, err = svc.List(ctx, invited, "", 10)
	if err != nil || page.Items[0].Author.Username == "" || page.Items[0].Author.RoleLevel == nil {
		t.Fatalf("an invited account lost the author's identity: %+v %v", page.Items[0].Author, err)
	}
}

func TestOnlyMembersSeeWhoIsInAGroup(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(quiet())
	svc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	svc.SetProfileLoader(func(_ context.Context, id uuid.UUID) (any, bool) { return id, true })

	owner := testdb.SeedUser(t, pool, "owner_"+uuid.NewString()[:6], 3)
	stranger := testdb.SeedUser(t, pool, "stranger_"+uuid.NewString()[:6], 5)
	ceo := testdb.SeedUser(t, pool, "ceo_"+uuid.NewString()[:6], 1)

	public, err := svc.Create(ctx, groups.CreateInput{Name: "Публичное", Kind: groups.KindCommunity, IsPublic: true},
		groups.ActorMeta{UserID: owner, RoleLevel: 3})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.ListMembers(ctx, public.ID, groups.ActorMeta{UserID: stranger, RoleLevel: 5}); !errors.Is(err, groups.ErrNotMember) {
		t.Fatalf("a non-member listed a community's members: %v", err)
	}
	if list, err := svc.ListMembers(ctx, public.ID, groups.ActorMeta{UserID: owner, RoleLevel: 3}); err != nil || len(list) != 1 {
		t.Fatalf("a member could not list members: %v %d", err, len(list))
	}
	if _, err := svc.ListMembers(ctx, public.ID, groups.ActorMeta{UserID: ceo, RoleLevel: 1}); err != nil {
		t.Fatalf("the CEO, who moderates, could not list members: %v", err)
	}
}

func TestPresenceFollowsOnlyChatPartners(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	me := testdb.SeedUser(t, pool, "me_"+uuid.NewString()[:6], 0)
	partner := testdb.SeedUser(t, pool, "partner_"+uuid.NewString()[:6], 5)
	boss := testdb.SeedUser(t, pool, "boss_"+uuid.NewString()[:6], 1)
	if _, err := pool.Exec(ctx, `INSERT INTO private_chats (user_a_id, user_b_id, initiated_by) VALUES ($1, $2, $1)`, me, partner); err != nil {
		t.Fatal(err)
	}

	allowed, err := chatPartnersOnly(pool)(ctx, me, []uuid.UUID{partner, boss, uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed) != 1 || allowed[0] != partner {
		t.Fatalf("presence allowed for %v, want only the chat partner %v", allowed, partner)
	}
}
