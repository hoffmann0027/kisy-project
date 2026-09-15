//go:build integration

package groups_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

func newGroups(t *testing.T) (*groups.Service, *pgxpool.Pool) {
	pool := testdb.New(t)
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	return svc, pool
}

func actor(id uuid.UUID, level int) groups.ActorMeta {
	return groups.ActorMeta{UserID: id, RoleLevel: level}
}

func TestCreateClearanceCap(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5)

	if _, err := svc.Create(ctx, groups.CreateInput{Name: "A", MinRoleLevel: 5}, actor(mgr, 5)); err != nil {
		t.Fatalf("create at own level: %v", err)
	}
	if _, err := svc.Create(ctx, groups.CreateInput{Name: "B", MinRoleLevel: 8}, actor(mgr, 5)); err != nil {
		t.Fatalf("create weaker: %v", err)
	}
	if _, err := svc.Create(ctx, groups.CreateInput{Name: "C", MinRoleLevel: 3}, actor(mgr, 5)); !errors.Is(err, groups.ErrLevelTooHigh) {
		t.Fatalf("create above own level: got %v, want ErrLevelTooHigh", err)
	}
}

func TestVisibilityMasking(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	ceo := testdb.SeedUser(t, pool, "ceo", 1)
	low := testdb.SeedUser(t, pool, "low", 10)

	// CEO creates a high-clearance group (min level 2).
	g, err := svc.Create(ctx, groups.CreateInput{Name: "Exec", MinRoleLevel: 2}, actor(ceo, 1))
	if err != nil {
		t.Fatal(err)
	}

	// CEO sees it; the low user does not.
	ceoVisible, _ := svc.ListVisible(ctx, actor(ceo, 1))
	if !containsGroup(ceoVisible, g.ID) {
		t.Fatal("CEO should see the group")
	}
	lowVisible, _ := svc.ListVisible(ctx, actor(low, 10))
	if containsGroup(lowVisible, g.ID) {
		t.Fatal("low-clearance user must not see a higher group")
	}

	// Get by id is masked as not-found for the low user.
	if _, err := svc.Get(ctx, g.ID, actor(low, 10)); !errors.Is(err, groups.ErrNotFound) {
		t.Fatalf("hidden Get: got %v, want ErrNotFound", err)
	}
}

func TestDeletePermissions(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	ceo := testdb.SeedUser(t, pool, "ceo", 1)
	founder := testdb.SeedUser(t, pool, "founder", 5)
	low := testdb.SeedUser(t, pool, "low", 10)

	mk := func(name string) uuid.UUID {
		g, err := svc.Create(ctx, groups.CreateInput{Name: name, MinRoleLevel: 5}, actor(founder, 5))
		if err != nil {
			t.Fatal(err)
		}
		return g.ID
	}

	// A low, unrelated user cannot see the group → delete is masked as 404.
	g1 := mk("g1")
	if err := svc.Delete(ctx, g1, actor(low, 10)); !errors.Is(err, groups.ErrNotFound) {
		t.Fatalf("hidden delete: got %v, want ErrNotFound", err)
	}

	// The founder can delete their own group.
	if err := svc.Delete(ctx, g1, actor(founder, 5)); err != nil {
		t.Fatalf("founder delete: %v", err)
	}

	// The CEO can delete any group.
	g2 := mk("g2")
	if err := svc.Delete(ctx, g2, actor(ceo, 1)); err != nil {
		t.Fatalf("CEO delete: %v", err)
	}
	if _, err := svc.Get(ctx, g2, actor(ceo, 1)); !errors.Is(err, groups.ErrNotFound) {
		t.Fatalf("group should be gone: %v", err)
	}
}

func TestSetMinRoleLevelPermissions(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	ceo := testdb.SeedUser(t, pool, "ceo", 1)
	founder := testdb.SeedUser(t, pool, "founder", 5)
	low := testdb.SeedUser(t, pool, "low", 10)

	g, err := svc.Create(ctx, groups.CreateInput{Name: "Team", MinRoleLevel: 5}, actor(founder, 5))
	if err != nil {
		t.Fatal(err)
	}

	// A low, unrelated user cannot see the group → change is masked as 404.
	if _, err := svc.SetMinRoleLevel(ctx, g.ID, 8, actor(low, 10)); !errors.Is(err, groups.ErrNotFound) {
		t.Fatalf("hidden change: got %v, want ErrNotFound", err)
	}
	// The founder can see it but is not the CEO → forbidden.
	if _, err := svc.SetMinRoleLevel(ctx, g.ID, 8, actor(founder, 5)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("founder change: got %v, want ErrForbidden", err)
	}

	// The CEO widens the group's audience to level 8.
	updated, err := svc.SetMinRoleLevel(ctx, g.ID, 8, actor(ceo, 1))
	if err != nil {
		t.Fatalf("CEO change: %v", err)
	}
	if updated.MinRoleLevel != 8 {
		t.Fatalf("min level not applied: got %d, want 8", updated.MinRoleLevel)
	}

	// The level-8 user can now see the group; tightening back to 4 hides it.
	if !containsGroup(mustList(t, svc, low, 8), g.ID) {
		t.Fatal("level-8 user should now see the widened group")
	}
	if _, err := svc.SetMinRoleLevel(ctx, g.ID, 4, actor(ceo, 1)); err != nil {
		t.Fatalf("CEO tighten: %v", err)
	}
	if containsGroup(mustList(t, svc, low, 8), g.ID) {
		t.Fatal("level-8 user must not see the tightened group")
	}
}

func mustList(t *testing.T, svc *groups.Service, id uuid.UUID, level int) []groups.Group {
	t.Helper()
	list, err := svc.ListVisible(context.Background(), actor(id, level))
	if err != nil {
		t.Fatal(err)
	}
	return list
}

// --- Stage N: access policies, self-join, join requests, roles ---

func dirHas(dir []groups.DirectoryEntry, id uuid.UUID) bool {
	for i := range dir {
		if dir[i].ID == id {
			return true
		}
	}
	return false
}

func TestJoinPoliciesAndClearance(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5)  // founder (owner)
	emp := testdb.SeedUser(t, pool, "emp", 8)  // may join (group min 8)
	low := testdb.SeedUser(t, pool, "low", 10) // too weak to see the group

	g, err := svc.Create(ctx, groups.CreateInput{Name: "G", MinRoleLevel: 8}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if g.JoinPolicy != groups.PolicyJoinRequest || g.PostPolicy != groups.PolicyPostAll {
		t.Fatalf("default policies: got %s/%s, want request/all", g.JoinPolicy, g.PostPolicy)
	}

	// Clearance invariant: a weaker user cannot see it in the directory, and
	// cannot join or apply — masked as not-found.
	if dir, _ := svc.Directory(ctx, actor(low, 10)); dirHas(dir, g.ID) {
		t.Fatal("low-clearance user must not see the group in the directory")
	}
	if _, err := svc.Join(ctx, g.ID, actor(low, 10)); !errors.Is(err, groups.ErrNotFound) {
		t.Fatalf("low join: got %v, want ErrNotFound", err)
	}

	// emp sees it and applies → pending; a second apply is idempotent.
	if dir, _ := svc.Directory(ctx, actor(emp, 8)); !dirHas(dir, g.ID) {
		t.Fatal("emp should see the group in the directory")
	}
	res, err := svc.Join(ctx, g.ID, actor(emp, 8))
	if err != nil || res.Status != "pending" {
		t.Fatalf("emp apply: %v, %+v", err, res)
	}
	if _, err := svc.Join(ctx, g.ID, actor(emp, 8)); err != nil {
		t.Fatalf("second apply should be idempotent: %v", err)
	}
	if member, _ := svc.IsMember(ctx, g.ID, emp); member {
		t.Fatal("applicant must not be a member yet")
	}

	// Owner approves → member; group leaves emp's directory.
	if err := svc.ApproveRequest(ctx, g.ID, emp, 8, actor(mgr, 5)); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if member, _ := svc.IsMember(ctx, g.ID, emp); !member {
		t.Fatal("approved applicant should be a member")
	}
	if dir, _ := svc.Directory(ctx, actor(emp, 8)); dirHas(dir, g.ID) {
		t.Fatal("a member must not see the group in the directory")
	}

	// Switch to open → another cleared user joins instantly.
	if _, err := svc.SetPolicies(ctx, g.ID, groups.PolicyJoinOpen, groups.PolicyPostAll, actor(mgr, 5)); err != nil {
		t.Fatalf("set policies: %v", err)
	}
	emp2 := testdb.SeedUser(t, pool, "emp2", 8)
	res, err = svc.Join(ctx, g.ID, actor(emp2, 8))
	if err != nil || !res.Joined {
		t.Fatalf("open self-join: %v, %+v", err, res)
	}
}

func TestRejectAndNoDoublePending(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5)
	emp := testdb.SeedUser(t, pool, "emp", 8)
	g, err := svc.Create(ctx, groups.CreateInput{Name: "R", MinRoleLevel: 8}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Join(ctx, g.ID, actor(emp, 8)); err != nil {
		t.Fatal(err)
	}
	if err := svc.RejectRequest(ctx, g.ID, emp, actor(mgr, 5)); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if member, _ := svc.IsMember(ctx, g.ID, emp); member {
		t.Fatal("rejected applicant must not be a member")
	}
	// After rejection the user may apply again (a new pending row).
	if res, err := svc.Join(ctx, g.ID, actor(emp, 8)); err != nil || res.Status != "pending" {
		t.Fatalf("re-apply after reject: %v, %+v", err, res)
	}
}

func TestPostPolicyEditors(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5) // founder/owner
	emp := testdb.SeedUser(t, pool, "emp", 8) // member
	g, err := svc.Create(ctx, groups.CreateInput{Name: "Chan", MinRoleLevel: 8}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetPolicies(ctx, g.ID, groups.PolicyJoinOpen, groups.PolicyPostEditors, actor(mgr, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Join(ctx, g.ID, actor(emp, 8)); err != nil {
		t.Fatal(err)
	}

	// Plain member cannot post; owner can.
	if err := svc.EnsureCanPost(ctx, g.ID, actor(emp, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("member post: got %v, want ErrForbidden", err)
	}
	if err := svc.EnsureCanPost(ctx, g.ID, actor(mgr, 5)); err != nil {
		t.Fatalf("owner post: %v", err)
	}
	// Promote member to editor → may post.
	if err := svc.SetMemberRole(ctx, g.ID, emp, groups.RoleEditor, actor(mgr, 5)); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if err := svc.EnsureCanPost(ctx, g.ID, actor(emp, 8)); err != nil {
		t.Fatalf("editor post: %v", err)
	}
}

func TestPolicyChangePermissions(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5)
	emp := testdb.SeedUser(t, pool, "emp", 8)
	g, err := svc.Create(ctx, groups.CreateInput{Name: "P", MinRoleLevel: 8}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetPolicies(ctx, g.ID, groups.PolicyJoinOpen, groups.PolicyPostAll, actor(mgr, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Join(ctx, g.ID, actor(emp, 8)); err != nil {
		t.Fatal(err)
	}
	// A plain member cannot change policies.
	if _, err := svc.SetPolicies(ctx, g.ID, groups.PolicyJoinRequest, groups.PolicyPostAll, actor(emp, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("member policy change: got %v, want ErrForbidden", err)
	}
	// A member also cannot view pending requests.
	if _, err := svc.ListRequests(ctx, g.ID, actor(emp, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("member list requests: got %v, want ErrForbidden", err)
	}
}

// Seeing a group used to be enough to add anyone to it, yourself included —
// which made a request-only join policy optional.
func TestAddMemberIsTheOwnersCall(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5)
	emp := testdb.SeedUser(t, pool, "emp", 8)
	other := testdb.SeedUser(t, pool, "other", 8)

	g, err := svc.Create(ctx, groups.CreateInput{Name: "Closed", MinRoleLevel: 8}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if g.JoinPolicy != groups.PolicyJoinRequest {
		t.Fatalf("precondition: want a request-only group, got %s", g.JoinPolicy)
	}

	if err := svc.AddMember(ctx, g.ID, emp, 8, actor(emp, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("adding yourself past the join policy: got %v, want ErrForbidden", err)
	}
	if member, _ := svc.IsMember(ctx, g.ID, emp); member {
		t.Fatal("a refused self-add must not leave a membership behind")
	}
	if err := svc.AddMember(ctx, g.ID, other, 8, actor(mgr, 5)); err != nil {
		t.Fatalf("owner adds a member: %v", err)
	}
}

func TestCommunityCannotEnrolAnyone(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	ceo := testdb.SeedUser(t, pool, "ceo", 1)
	mgr := testdb.SeedUser(t, pool, "mgr", 5)
	emp := testdb.SeedUser(t, pool, "emp", 8)

	c, err := svc.Create(ctx, groups.CreateInput{Name: "Wall", Kind: groups.KindCommunity}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	for name, who := range map[string]groups.ActorMeta{"owner": actor(mgr, 5), "CEO": actor(ceo, 1)} {
		if err := svc.AddMember(ctx, c.ID, emp, 8, who); !errors.Is(err, groups.ErrForbidden) {
			t.Fatalf("%s enrols a user into a community: got %v, want ErrForbidden", name, err)
		}
	}
	if member, _ := svc.IsMember(ctx, c.ID, emp); member {
		t.Fatal("nobody may be put into a community without joining it themselves")
	}
}

func TestCommunityWorkspaceIsForEditors(t *testing.T) {
	svc, pool := newGroups(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5)
	reader := testdb.SeedUser(t, pool, "reader", 8)
	stranger := testdb.SeedUser(t, pool, "stranger", 8)

	c, err := svc.Create(ctx, groups.CreateInput{Name: "Wall", Kind: groups.KindCommunity, IsPublic: true}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetPolicies(ctx, c.ID, groups.PolicyJoinOpen, groups.PolicyPostEditors, actor(mgr, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Join(ctx, c.ID, actor(reader, 8)); err != nil {
		t.Fatal(err)
	}

	if err := svc.EnsureWorkspace(ctx, c.ID, actor(mgr, 5)); err != nil {
		t.Fatalf("owner opens the board: %v", err)
	}
	if err := svc.EnsureWorkspace(ctx, c.ID, actor(reader, 8)); !errors.Is(err, groups.ErrForbidden) {
		t.Fatalf("plain reader opens the board: got %v, want ErrForbidden", err)
	}
	if err := svc.EnsureWorkspace(ctx, c.ID, actor(stranger, 8)); !errors.Is(err, groups.ErrNotMember) {
		t.Fatalf("non-member opens the board: got %v, want ErrNotMember", err)
	}
	vs, err := svc.Viewer(ctx, c.ID, actor(reader, 8))
	if err != nil {
		t.Fatal(err)
	}
	if vs.CanUseWorkspace {
		t.Fatal("the client must not be told to show a reader the board tab")
	}
	if ok, _ := svc.HasWorkspace(ctx, c.ID, reader, 8); ok {
		t.Fatal("a card must not be assignable to someone who cannot open the board")
	}

	if err := svc.SetMemberRole(ctx, c.ID, reader, groups.RoleEditor, actor(mgr, 5)); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureWorkspace(ctx, c.ID, actor(reader, 8)); err != nil {
		t.Fatalf("promoted editor opens the board: %v", err)
	}

	// An ordinary group keeps its board open to every member.
	g, err := svc.Create(ctx, groups.CreateInput{Name: "Team", MinRoleLevel: 8}, actor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(ctx, g.ID, stranger, 8, actor(mgr, 5)); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureWorkspace(ctx, g.ID, actor(stranger, 8)); err != nil {
		t.Fatalf("group member opens the group's board: %v", err)
	}
}

func containsGroup(list []groups.Group, id uuid.UUID) bool {
	for i := range list {
		if list[i].ID == id {
			return true
		}
	}
	return false
}
