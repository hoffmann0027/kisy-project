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

func containsGroup(list []groups.Group, id uuid.UUID) bool {
	for i := range list {
		if list[i].ID == id {
			return true
		}
	}
	return false
}
