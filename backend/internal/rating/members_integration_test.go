//go:build integration

package rating_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/rating"
)

// A project has members who answer for it: they record its money, and the
// row names them. Levels 1–4 create projects of their own and run them —
// tasks, members, money, the level, deletion; the CEO runs every project.
// Everyone else takes tasks and moves them, as before.

type membersFixture struct {
	ctx                               context.Context
	svc                               *rating.Service
	ceo, lead, otherLead, dev, junior rating.Actor
	levels                            map[uuid.UUID]int
}

func newMembersFixture(t *testing.T) membersFixture {
	t.Helper()
	pool := testdb.New(t)
	svc := rating.NewService(pool, rating.NewPostgresRepository())
	suffix := uuid.NewString()[:8]
	mk := func(name string, level int) rating.Actor {
		return rating.Actor{UserID: testdb.SeedUser(t, pool, name+"_"+suffix, level), RoleLevel: level}
	}
	f := membersFixture{ctx: context.Background(), svc: svc, levels: map[uuid.UUID]int{}}
	f.ceo = mk("ceo", 1)
	f.lead = mk("lead", 4)
	f.otherLead = mk("lead2", 3)
	f.dev = mk("dev", 8)
	f.junior = mk("junior", 9)
	for _, a := range []rating.Actor{f.ceo, f.lead, f.otherLead, f.dev, f.junior} {
		f.levels[a.UserID] = a.RoleLevel
	}
	svc.SetUserLookup(func(_ context.Context, id uuid.UUID) (int, bool) {
		lvl, ok := f.levels[id]
		return lvl, ok
	})
	return f
}

func (f membersFixture) project(t *testing.T, by rating.Actor, minLevel int) uuid.UUID {
	t.Helper()
	id, err := f.svc.CreateProject(f.ctx, rating.CreateProjectInput{Title: "P", Difficulty: "medium", MinLevel: minLevel}, by)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return id
}

func (f membersFixture) membersOf(t *testing.T, id uuid.UUID, level int) []uuid.UUID {
	t.Helper()
	board, err := f.svc.Board(f.ctx, level)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range board.Projects {
		if p.ID == id {
			out := make([]uuid.UUID, 0, len(p.Members))
			for _, m := range p.Members {
				out = append(out, m.ID)
			}
			return out
		}
	}
	t.Fatalf("project %s is not on the board", id)
	return nil
}

func TestManagersCreateProjectsWithinTheirClearance(t *testing.T) {
	f := newMembersFixture(t)
	if _, err := f.svc.CreateProject(f.ctx, rating.CreateProjectInput{Title: "P", MinLevel: 8}, f.dev); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("level 8 creating a project: got %v, want ErrForbidden", err)
	}
	// Level 4 may not put a project at level 3: they could not see it.
	if _, err := f.svc.CreateProject(f.ctx, rating.CreateProjectInput{Title: "P", MinLevel: 3}, f.lead); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("level 4 creating a level-3 project: got %v, want ErrForbidden", err)
	}
	f.project(t, f.lead, 4)
	f.project(t, f.lead, 8)
}

func TestTheCreatorRunsTheProjectAndNobodyElseBut(t *testing.T) {
	f := newMembersFixture(t)
	id := f.project(t, f.lead, 8)

	taskID, err := f.svc.CreateTask(f.ctx, id, "task", f.lead)
	if err != nil {
		t.Fatalf("creator adding a task: %v", err)
	}
	if _, err := f.svc.CreateTask(f.ctx, id, "task", f.otherLead); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("another manager adding a task: got %v, want ErrForbidden", err)
	}
	if _, err := f.svc.CreateTask(f.ctx, id, "task", f.dev); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("a member-to-be adding a task: got %v, want ErrForbidden", err)
	}
	if err := f.svc.DeleteTask(f.ctx, taskID, f.otherLead); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("another manager deleting a task: got %v, want ErrForbidden", err)
	}
	if err := f.svc.DeleteTask(f.ctx, taskID, f.ceo); err != nil {
		t.Fatalf("the CEO deleting a task: %v", err)
	}
	if err := f.svc.SetProjectLevel(f.ctx, id, 3, f.lead); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("creator raising the level above their own: got %v, want ErrForbidden", err)
	}
	if err := f.svc.SetProjectLevel(f.ctx, id, 6, f.lead); err != nil {
		t.Fatalf("creator changing the level: %v", err)
	}
	if err := f.svc.DeleteProject(f.ctx, id, f.otherLead); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("another manager deleting the project: got %v, want ErrForbidden", err)
	}
	if err := f.svc.DeleteProject(f.ctx, id, f.lead); err != nil {
		t.Fatalf("creator deleting the project: %v", err)
	}
}

func TestMembersRecordMoneyAndAreOnTheRow(t *testing.T) {
	f := newMembersFixture(t)
	id := f.project(t, f.lead, 8)
	money := rating.FinanceInput{IncomeKopecks: 10_000}

	// Not yet a member: no.
	if err := f.svc.AddFinance(f.ctx, id, money, f.dev); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("a non-member recording money: got %v, want ErrForbidden", err)
	}
	// Only the creator (or the CEO) changes the list.
	if err := f.svc.AddMember(f.ctx, id, f.dev.UserID, f.otherLead); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("another manager adding a member: got %v, want ErrForbidden", err)
	}
	if err := f.svc.AddMember(f.ctx, id, f.dev.UserID, f.lead); err != nil {
		t.Fatalf("creator adding a member: %v", err)
	}
	// Twice is harmless; a stranger to the directory is not found.
	if err := f.svc.AddMember(f.ctx, id, f.dev.UserID, f.lead); err != nil {
		t.Fatalf("adding twice: %v", err)
	}
	if err := f.svc.AddMember(f.ctx, id, uuid.New(), f.lead); !errors.Is(err, rating.ErrNotFound) {
		t.Fatalf("adding an unknown user: got %v, want ErrNotFound", err)
	}
	// Level 9 cannot see a level-8 project, so cannot be on it.
	if err := f.svc.AddMember(f.ctx, id, f.junior.UserID, f.lead); !errors.Is(err, rating.ErrOutOfReach) {
		t.Fatalf("adding someone below the threshold: got %v, want ErrOutOfReach", err)
	}
	if got := f.membersOf(t, id, 8); len(got) != 1 || got[0] != f.dev.UserID {
		t.Fatalf("members = %v, want [dev]", got)
	}

	// Now a member: yes, and the row carries their name.
	if err := f.svc.AddFinance(f.ctx, id, money, f.dev); err != nil {
		t.Fatalf("a member recording money: %v", err)
	}
	a, err := f.svc.Analytics(f.ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Recent) == 0 || a.Recent[0].ProjectID != id || a.Recent[0].AuthorName == "" {
		t.Fatalf("the ledger does not name who recorded: %+v", a.Recent)
	}
	// The creator and the CEO record without being members.
	if err := f.svc.AddFinance(f.ctx, id, money, f.lead); err != nil {
		t.Fatalf("the creator recording money: %v", err)
	}
	if err := f.svc.AddFinance(f.ctx, id, money, f.ceo); err != nil {
		t.Fatalf("the CEO recording money: %v", err)
	}

	// Off the list: no again.
	if err := f.svc.RemoveMember(f.ctx, id, f.dev.UserID, f.dev); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("a member removing themselves: got %v, want ErrForbidden", err)
	}
	if err := f.svc.RemoveMember(f.ctx, id, f.dev.UserID, f.ceo); err != nil {
		t.Fatalf("the CEO removing a member: %v", err)
	}
	if err := f.svc.RemoveMember(f.ctx, id, f.dev.UserID, f.ceo); !errors.Is(err, rating.ErrNotFound) {
		t.Fatalf("removing twice: got %v, want ErrNotFound", err)
	}
	if err := f.svc.AddFinance(f.ctx, id, money, f.dev); !errors.Is(err, rating.ErrForbidden) {
		t.Fatalf("a removed member recording money: got %v, want ErrForbidden", err)
	}
	if got := f.membersOf(t, id, 8); len(got) != 0 {
		t.Fatalf("members = %v, want none", got)
	}
}
