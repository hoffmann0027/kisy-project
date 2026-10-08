//go:build integration

package groups_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

// A community could be joined but never left, and the people who run it could
// not show anyone the door. Members leave now; the founder, editors and
// moderators remove and ban those below them.

type community struct {
	svc                                   *groups.Service
	id                                    uuid.UUID
	founder, editor, moderator, a, b, ceo uuid.UUID
	told                                  map[uuid.UUID]int
	mu                                    sync.Mutex
}

func newCommunity(t *testing.T) *community {
	t.Helper()
	svc, pool := newGroups(t)
	ctx := context.Background()
	c := &community{svc: svc, told: map[uuid.UUID]int{}}
	svc.SetFormerMemberPublisher(func(userID, _ uuid.UUID) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.told[userID]++
	})
	c.ceo = testdb.SeedUser(t, pool, "the_ceo", 1)
	c.founder = testdb.SeedUser(t, pool, "founder", 5)
	c.editor = testdb.SeedUser(t, pool, "editor", 8)
	c.moderator = testdb.SeedUser(t, pool, "moderator", 8)
	c.a = testdb.SeedUser(t, pool, "reader_a", 8)
	c.b = testdb.SeedUser(t, pool, "reader_b", 8)

	g, err := svc.Create(ctx, groups.CreateInput{Name: "Vibe", Kind: groups.KindCommunity, IsPublic: true}, actor(c.founder, 5))
	if err != nil {
		t.Fatal(err)
	}
	c.id = g.ID
	if _, err := svc.SetPolicies(ctx, c.id, groups.PolicyJoinOpen, groups.PolicyPostEditors, actor(c.founder, 5)); err != nil {
		t.Fatal(err)
	}
	for _, u := range []uuid.UUID{c.editor, c.moderator, c.a, c.b} {
		if _, err := svc.Join(ctx, c.id, actor(u, 8)); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.SetMemberRole(ctx, c.id, c.editor, groups.RoleEditor, actor(c.founder, 5)); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetMemberRole(ctx, c.id, c.moderator, groups.RoleModerator, actor(c.founder, 5)); err != nil {
		t.Fatal(err)
	}
	return c
}

func (c *community) member(t *testing.T, u uuid.UUID) bool {
	t.Helper()
	ok, err := c.svc.IsMember(context.Background(), c.id, u)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestAMemberLeavesButTheFounderStays(t *testing.T) {
	c := newCommunity(t)
	ctx := context.Background()

	if err := c.svc.Leave(ctx, c.id, actor(c.a, 8)); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if c.member(t, c.a) {
		t.Fatal("still a member after leaving")
	}
	if c.told[c.a] != 1 {
		t.Fatal("the leaver's other devices were not told to drop the community")
	}
	if err := c.svc.Leave(ctx, c.id, actor(c.a, 8)); !errors.Is(err, groups.ErrNotMember) {
		t.Fatalf("leaving twice: got %v, want ErrNotMember", err)
	}
	// Leaving is not a ban: the way back in is still open.
	if _, err := c.svc.Join(ctx, c.id, actor(c.a, 8)); err != nil {
		t.Fatalf("rejoin after leaving: %v", err)
	}

	if err := c.svc.Leave(ctx, c.id, actor(c.founder, 5)); !errors.Is(err, groups.ErrFounderStays) {
		t.Fatalf("founder leaving: got %v, want ErrFounderStays", err)
	}
}

func TestOnlyAHigherRankRemoves(t *testing.T) {
	c := newCommunity(t)
	ctx := context.Background()
	remove := func(by uuid.UUID, level int, target uuid.UUID) error {
		return c.svc.RemoveMember(ctx, c.id, target, actor(by, level))
	}

	// A plain member removes no one.
	if err := remove(c.a, 8, c.b); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("member removing a member: got %v, want ErrForbidden", err)
	}
	// A moderator removes a member, but not an editor — nor another
	// moderator: equals do not remove each other.
	if err := remove(c.moderator, 8, c.editor); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("moderator removing an editor: got %v, want ErrForbidden", err)
	}
	if err := c.svc.SetMemberRole(ctx, c.id, c.b, groups.RoleModerator, actor(c.founder, 5)); err != nil {
		t.Fatal(err)
	}
	if err := remove(c.moderator, 8, c.b); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("moderator removing a moderator: got %v, want ErrForbidden", err)
	}
	if err := remove(c.moderator, 8, c.a); err != nil {
		t.Fatalf("moderator removing a member: %v", err)
	}
	if c.member(t, c.a) || c.told[c.a] != 1 {
		t.Fatal("the removed member is still in, or was not told")
	}
	// An editor removes a moderator; nobody removes the founder, the CEO
	// included, and nobody removes themselves (that is leaving).
	if err := remove(c.editor, 8, c.moderator); err != nil {
		t.Fatalf("editor removing a moderator: %v", err)
	}
	if err := remove(c.editor, 8, c.founder); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("editor removing the founder: got %v, want ErrForbidden", err)
	}
	if err := remove(c.ceo, 1, c.founder); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("CEO removing the founder: got %v, want ErrForbidden", err)
	}
	if err := remove(c.editor, 8, c.editor); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("removing oneself: got %v, want ErrForbidden", err)
	}
	// The founder and the CEO remove anyone else.
	if err := remove(c.ceo, 1, c.b); err != nil {
		t.Fatalf("CEO removing a member: %v", err)
	}
	if err := remove(c.founder, 5, c.editor); err != nil {
		t.Fatalf("founder removing an editor: %v", err)
	}
}

func TestABanKeepsThemOut(t *testing.T) {
	c := newCommunity(t)
	ctx := context.Background()

	if err := c.svc.Ban(ctx, c.id, c.a, actor(c.a, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("banning oneself: got %v, want ErrForbidden", err)
	}
	if err := c.svc.Ban(ctx, c.id, c.a, actor(c.moderator, 8)); err != nil {
		t.Fatalf("moderator banning a member: %v", err)
	}
	if c.member(t, c.a) || c.told[c.a] != 1 {
		t.Fatal("the banned member is still in, or was not told")
	}
	if _, err := c.svc.Join(ctx, c.id, actor(c.a, 8)); !errors.Is(err, groups.ErrBanned) {
		t.Fatalf("banned rejoining: got %v, want ErrBanned", err)
	}
	if dir, _ := c.svc.Directory(ctx, actor(c.a, 8)); dirHas(dir, c.id) {
		t.Fatal("a banned user is still offered the community to join")
	}

	// Only those who run it see the list and lift a ban.
	if _, err := c.svc.ListBans(ctx, c.id, actor(c.b, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("member listing bans: got %v, want ErrForbidden", err)
	}
	bans, err := c.svc.ListBans(ctx, c.id, actor(c.editor, 8))
	if err != nil {
		t.Fatal(err)
	}
	if len(bans) != 0 {
		// No profile loader in this test: entries are resolved to profiles,
		// and an unresolvable one is skipped rather than leaked as a bare id.
		t.Fatalf("bans without profiles: %v", bans)
	}
	if err := c.svc.Unban(ctx, c.id, c.a, actor(c.b, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("member lifting a ban: got %v, want ErrForbidden", err)
	}
	if err := c.svc.Unban(ctx, c.id, c.a, actor(c.editor, 8)); err != nil {
		t.Fatalf("editor lifting a ban: %v", err)
	}
	if err := c.svc.Unban(ctx, c.id, c.a, actor(c.editor, 8)); !errors.Is(err, groups.ErrNotFound) {
		t.Fatalf("lifting a ban twice: got %v, want ErrNotFound", err)
	}
	if _, err := c.svc.Join(ctx, c.id, actor(c.a, 8)); err != nil {
		t.Fatalf("rejoining after the ban was lifted: %v", err)
	}
}

// A ban reaches someone who only applied, and their request goes with it.
func TestBanningAnApplicantDropsTheRequest(t *testing.T) {
	c := newCommunity(t)
	ctx := context.Background()
	if _, err := c.svc.SetPolicies(ctx, c.id, groups.PolicyJoinRequest, groups.PolicyPostEditors, actor(c.founder, 5)); err != nil {
		t.Fatal(err)
	}
	if err := c.svc.Leave(ctx, c.id, actor(c.b, 8)); err != nil {
		t.Fatal(err)
	}
	if res, err := c.svc.Join(ctx, c.id, actor(c.b, 8)); err != nil || res.Status != "pending" {
		t.Fatalf("apply: %v %+v", err, res)
	}
	if err := c.svc.Ban(ctx, c.id, c.b, actor(c.founder, 5)); err != nil {
		t.Fatalf("banning an applicant: %v", err)
	}
	if err := c.svc.ApproveRequest(ctx, c.id, c.b, 8, actor(c.founder, 5)); !errors.Is(err, groups.ErrRequestNotFound) {
		t.Fatalf("approving a banned applicant: got %v, want ErrRequestNotFound", err)
	}
	if c.member(t, c.b) {
		t.Fatal("a banned applicant got in")
	}
}
