//go:build integration

package calendar_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/calendar"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

func setup(t *testing.T) (*calendar.Service, *groups.Service, *pgxpool.Pool) {
	pool := testdb.New(t)
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	gsvc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	csvc := calendar.NewService(pool, calendar.NewPostgresRepository(), rec, calendar.Access{
		EnsureMember: func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error {
			err := gsvc.EnsureMember(ctx, groupID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
			switch {
			case errors.Is(err, groups.ErrNotFound):
				return calendar.ErrNotFound
			case errors.Is(err, groups.ErrNotMember):
				return calendar.ErrForbidden
			default:
				return err
			}
		},
		IsFounder: gsvc.IsFounder,
	})
	return csvc, gsvc, pool
}

func cActor(id uuid.UUID, level int) calendar.Actor {
	return calendar.Actor{UserID: id, RoleLevel: level}
}
func gActor(id uuid.UUID, level int) groups.ActorMeta {
	return groups.ActorMeta{UserID: id, RoleLevel: level}
}

func TestCalendarAccessAndModeration(t *testing.T) {
	csvc, gsvc, pool := setup(t)
	ctx := context.Background()
	mgr := testdb.SeedUser(t, pool, "mgr", 5)   // founder/owner
	emp := testdb.SeedUser(t, pool, "emp", 5)   // member, event author
	emp2 := testdb.SeedUser(t, pool, "emp2", 5) // another member
	low := testdb.SeedUser(t, pool, "low", 10)  // uncleared outsider

	g, err := gsvc.Create(ctx, groups.CreateInput{Name: "G", MinRoleLevel: 5}, gActor(mgr, 5))
	if err != nil {
		t.Fatal(err)
	}
	if err := gsvc.AddMember(ctx, g.ID, emp, 5, gActor(mgr, 5)); err != nil {
		t.Fatal(err)
	}
	if err := gsvc.AddMember(ctx, g.ID, emp2, 5, gActor(mgr, 5)); err != nil {
		t.Fatal(err)
	}

	from := time.Now().Add(-24 * time.Hour)
	to := from.Add(30 * 24 * time.Hour)

	// Uncleared outsider cannot even see the calendar → ErrNotFound.
	if _, err := csvc.List(ctx, g.ID, from, to, cActor(low, 10)); !errors.Is(err, calendar.ErrNotFound) {
		t.Fatalf("outsider list: got %v, want ErrNotFound", err)
	}

	// A member creates an event.
	ev, err := csvc.Create(ctx, g.ID, calendar.Input{Title: "Standup", StartsAt: from.Add(time.Hour), Color: "blue"}, cActor(emp, 5))
	if err != nil {
		t.Fatalf("member create: %v", err)
	}

	// Another member cannot edit or delete someone else's event.
	if _, err := csvc.Update(ctx, ev.ID, calendar.Input{Title: "Hijack", StartsAt: ev.StartsAt, Color: "red"}, cActor(emp2, 5)); !errors.Is(err, calendar.ErrForbidden) {
		t.Fatalf("non-owner update: got %v, want ErrForbidden", err)
	}
	if err := csvc.Delete(ctx, ev.ID, cActor(emp2, 5)); !errors.Is(err, calendar.ErrForbidden) {
		t.Fatalf("non-owner delete: got %v, want ErrForbidden", err)
	}

	// The group founder (owner) may moderate it.
	if _, err := csvc.Update(ctx, ev.ID, calendar.Input{Title: "Moved", StartsAt: ev.StartsAt, Color: "green"}, cActor(mgr, 5)); err != nil {
		t.Fatalf("founder update: %v", err)
	}

	// The author sees the event in the month list and may delete it.
	view, err := csvc.List(ctx, g.ID, from, to, cActor(emp, 5))
	if err != nil || len(view.Events) != 1 {
		t.Fatalf("member list: %v, events=%d", err, len(view.Events))
	}
	if err := csvc.Delete(ctx, ev.ID, cActor(emp, 5)); err != nil {
		t.Fatalf("author delete: %v", err)
	}
}
