//go:build integration

package disappear_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/disappear"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

// Audit A-14: a chat's timer decides how long everyone's next messages live,
// so changing it is writing. A plain reader of an editors-only channel could
// set it to 30 seconds and make the editors' next posts vanish.
func TestOnlyThoseWhoMayWriteCanChangeAGroupTimer(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))

	owner := testdb.SeedUser(t, pool, "owner", 3)
	reader := testdb.SeedUser(t, pool, "reader", 5)
	groupsSvc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	g, err := groupsSvc.Create(ctx, groups.CreateInput{Name: "channel", MinRoleLevel: 5}, groups.ActorMeta{UserID: owner, RoleLevel: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE groups SET post_policy = 'editors' WHERE id = $1`, g.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)`, g.ID, reader); err != nil {
		t.Fatal(err)
	}

	member := func(ctx context.Context, _ string, chatID, actorID uuid.UUID, level int) error {
		return groupsSvc.EnsureMember(ctx, chatID, groups.ActorMeta{UserID: actorID, RoleLevel: level})
	}
	svc := disappear.NewService(pool, disappear.NewPostgresRepository(), member, rec)
	// As cmd/server wires it.
	svc.SetWriteAuthorizer(func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, level int) error {
		if chatType != "group" {
			return nil
		}
		return groupsSvc.EnsureCanPost(ctx, chatID, groups.ActorMeta{UserID: actorID, RoleLevel: level})
	})

	thirty := int64(30)
	if _, err := svc.Set(ctx, "group", g.ID, &thirty, disappear.Actor{UserID: reader, RoleLevel: 5}); !errors.Is(err, disappear.ErrForbidden) {
		t.Fatalf("a reader set the timer of an editors-only group: %v", err)
	}
	// Reading it is still allowed.
	if _, err := svc.Get(ctx, "group", g.ID, disappear.Actor{UserID: reader, RoleLevel: 5}); err != nil {
		t.Fatalf("a reader could not read the timer: %v", err)
	}
	if _, err := svc.Set(ctx, "group", g.ID, &thirty, disappear.Actor{UserID: owner, RoleLevel: 3}); err != nil {
		t.Fatalf("the owner could not set the timer: %v", err)
	}
}
