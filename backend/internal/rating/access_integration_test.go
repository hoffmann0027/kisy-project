//go:build integration

package rating_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/rating"
)

// Audit A-12: a project has a clearance threshold, and its board is hidden
// from anyone below it — but assigning yourself one of its tasks, moving the
// progress and returning it checked nothing. A level-5 employee could close
// tasks of a project they were never allowed to see, which moves that
// project's rating and its money.

type ratingFixture struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	svc    *rating.Service
	hidden uuid.UUID // task of a project above the outsider's clearance
	open   uuid.UUID // task of a project they may see
	// outsider is level 5; the hidden project requires level 3 or stronger.
	outsider rating.Actor
	insider  rating.Actor
}

func newRatingFixture(t *testing.T) ratingFixture {
	t.Helper()
	pool := testdb.New(t)
	ctx := context.Background()
	repo := rating.NewPostgresRepository()
	svc := rating.NewService(pool, repo)
	suffix := uuid.NewString()[:8]

	f := ratingFixture{ctx: ctx, pool: pool, svc: svc}
	f.outsider = rating.Actor{UserID: testdb.SeedUser(t, pool, "out_"+suffix, 5), RoleLevel: 5}
	f.insider = rating.Actor{UserID: testdb.SeedUser(t, pool, "in_"+suffix, 3), RoleLevel: 3}

	project := func(minLevel int) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `
			INSERT INTO rating_projects (title, difficulty, min_level, created_by)
			VALUES ($1, 'medium', $2, $3) RETURNING id`,
			"P"+uuid.NewString()[:6], minLevel, f.insider.UserID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	task := func(projectID uuid.UUID) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `
			INSERT INTO rating_tasks (project_id, title, status)
			VALUES ($1, $2, 'backlog') RETURNING id`, projectID, "T"+uuid.NewString()[:6]).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	f.hidden = task(project(3))
	f.open = task(project(7))
	return f
}

func (f ratingFixture) taskState(t *testing.T, id uuid.UUID) (assignee *uuid.UUID, status string, progress int) {
	t.Helper()
	if err := f.pool.QueryRow(f.ctx,
		`SELECT assignee_id, status, progress FROM rating_tasks WHERE id = $1`, id).
		Scan(&assignee, &status, &progress); err != nil {
		t.Fatal(err)
	}
	return assignee, status, progress
}

func TestATaskOfAHiddenProjectCannotBeTaken(t *testing.T) {
	f := newRatingFixture(t)

	if err := f.svc.AssignSelf(f.ctx, f.hidden, f.outsider); err == nil {
		t.Fatal("a task of a project above the caller's clearance was assigned to them")
	}
	assignee, status, _ := f.taskState(t, f.hidden)
	if assignee != nil || status != "backlog" {
		t.Fatalf("the hidden task changed: assignee=%v status=%s", assignee, status)
	}

	// The same person takes a task of a project they may see.
	if err := f.svc.AssignSelf(f.ctx, f.open, f.outsider); err != nil {
		t.Fatalf("an ordinary task: %v", err)
	}
}

func TestProgressAndReturnRespectTheProjectThreshold(t *testing.T) {
	f := newRatingFixture(t)
	// The insider legitimately takes the hidden project's task.
	if err := f.svc.AssignSelf(f.ctx, f.hidden, f.insider); err != nil {
		t.Fatal(err)
	}

	// The outsider is not its assignee and cannot see the project at all.
	if err := f.svc.SetProgress(f.ctx, f.hidden, 100, f.outsider); err == nil {
		t.Fatal("an outsider set progress on a hidden project's task")
	}
	if err := f.svc.ReturnTask(f.ctx, f.hidden, f.outsider); err == nil {
		t.Fatal("an outsider returned a hidden project's task to the backlog")
	}
	assignee, status, progress := f.taskState(t, f.hidden)
	if assignee == nil || *assignee != f.insider.UserID || status != "in_progress" || progress != 0 {
		t.Fatalf("the hidden task changed: assignee=%v status=%s progress=%d", assignee, status, progress)
	}

	// Its own assignee still works normally.
	if err := f.svc.SetProgress(f.ctx, f.hidden, 50, f.insider); err != nil {
		t.Fatalf("the assignee: %v", err)
	}
	if _, _, progress = f.taskState(t, f.hidden); progress != 50 {
		t.Fatalf("progress = %d, want 50", progress)
	}
}

// A hidden project must not be discoverable through the error either: a task
// that exists behind a threshold answers like one that does not exist.
func TestAHiddenTaskIsNotFoundRatherThanForbidden(t *testing.T) {
	f := newRatingFixture(t)
	err := f.svc.ReturnTask(f.ctx, f.hidden, f.outsider)
	if !errors.Is(err, rating.ErrNotFound) {
		t.Fatalf("returning a hidden task: %v, want ErrNotFound", err)
	}
}
