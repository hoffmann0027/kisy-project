//go:build integration

package boards_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/boards"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

// setup wires a real groups service into the boards access hooks against a
// fresh database, then creates a group with a founder and one extra member.
func setup(t *testing.T) (svc *boards.Service, ctx context.Context, groupID, founder, member uuid.UUID) {
	pool := testdb.New(t)
	ctx = context.Background()
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))

	gsvc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	founder = testdb.SeedUser(t, pool, "founder", 3)
	member = testdb.SeedUser(t, pool, "member", 5)

	g, err := gsvc.Create(ctx, groups.CreateInput{Name: "Team", MinRoleLevel: 6}, groups.ActorMeta{UserID: founder, RoleLevel: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := gsvc.AddMember(ctx, g.ID, member, 5, groups.ActorMeta{UserID: founder, RoleLevel: 3}); err != nil {
		t.Fatal(err)
	}

	svc = boards.NewService(pool, boards.NewPostgresRepository(), boards.Access{
		EnsureActorMember: func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error {
			return gsvc.EnsureMember(ctx, groupID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
		},
		IsFounder: gsvc.IsFounder,
		IsMember:  gsvc.IsMember,
	})
	return svc, ctx, g.ID, founder, member
}

func founderActor(id uuid.UUID) boards.Actor { return boards.Actor{UserID: id, RoleLevel: 3} }
func memberActor(id uuid.UUID) boards.Actor  { return boards.Actor{UserID: id, RoleLevel: 5} }

func TestBoardFounderOnlyCreate(t *testing.T) {
	svc, ctx, groupID, founder, member := setup(t)

	// No board yet.
	if _, err := svc.Get(ctx, groupID, founderActor(founder)); !errors.Is(err, boards.ErrNotFound) {
		t.Fatalf("expected ErrNotFound before creation, got %v", err)
	}
	// A non-founder member cannot create the board.
	if _, err := svc.Create(ctx, groupID, "B", memberActor(member)); !errors.Is(err, boards.ErrForbidden) {
		t.Fatalf("member create: got %v, want ErrForbidden", err)
	}
	// The founder can, and it seeds three columns.
	board, err := svc.Create(ctx, groupID, "Sprint", founderActor(founder))
	if err != nil {
		t.Fatalf("founder create: %v", err)
	}
	if len(board.Columns) != 3 {
		t.Fatalf("expected 3 seeded columns, got %d", len(board.Columns))
	}
	// A second board is rejected.
	if _, err := svc.Create(ctx, groupID, "B2", founderActor(founder)); !errors.Is(err, boards.ErrBoardExists) {
		t.Fatalf("duplicate board: got %v, want ErrBoardExists", err)
	}
}

func TestCardLifecycleAndMove(t *testing.T) {
	svc, ctx, groupID, founder, member := setup(t)
	board, err := svc.Create(ctx, groupID, "Sprint", founderActor(founder))
	if err != nil {
		t.Fatal(err)
	}
	todo := board.Columns[0].ID
	doing := board.Columns[1].ID

	// A member can create cards; the assignee must be a group member.
	label := "blue"
	c1, err := svc.CreateCard(ctx, todo, boards.CardInput{Title: "A", Label: &label, AssigneeID: &member}, memberActor(member))
	if err != nil {
		t.Fatalf("create card: %v", err)
	}
	if _, err := svc.CreateCard(ctx, todo, boards.CardInput{Title: "B"}, memberActor(member)); err != nil {
		t.Fatalf("create card B: %v", err)
	}
	// A non-member assignee is rejected.
	stranger := uuid.New()
	if _, err := svc.CreateCard(ctx, todo, boards.CardInput{Title: "X", AssigneeID: &stranger}, memberActor(member)); !errors.Is(err, boards.ErrInvalidInput) {
		t.Fatalf("non-member assignee: got %v, want ErrInvalidInput", err)
	}

	// Move card A to the "doing" column at index 0, then verify dense
	// positions in both columns.
	if err := svc.Move(ctx, c1.ID, doing, 0, memberActor(member)); err != nil {
		t.Fatalf("move: %v", err)
	}
	reloaded, err := svc.Get(ctx, groupID, memberActor(member))
	if err != nil {
		t.Fatal(err)
	}
	todoCol := findColumn(reloaded, todo)
	doingCol := findColumn(reloaded, doing)
	if len(doingCol.Cards) != 1 || doingCol.Cards[0].ID != c1.ID {
		t.Fatalf("card not moved into doing column")
	}
	if len(todoCol.Cards) != 1 || todoCol.Cards[0].Position != 0 {
		t.Fatalf("source column not re-sequenced: %+v", todoCol.Cards)
	}
}

func TestFounderOnlyColumns(t *testing.T) {
	svc, ctx, groupID, founder, member := setup(t)
	board, err := svc.Create(ctx, groupID, "Sprint", founderActor(founder))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddColumn(ctx, board.ID, "Review", memberActor(member)); !errors.Is(err, boards.ErrForbidden) {
		t.Fatalf("member add column: got %v, want ErrForbidden", err)
	}
	if err := svc.AddColumn(ctx, board.ID, "Review", founderActor(founder)); err != nil {
		t.Fatalf("founder add column: %v", err)
	}
}

func findColumn(b *boards.BoardDTO, id uuid.UUID) boards.ColumnDTO {
	for _, c := range b.Columns {
		if c.ID == id {
			return c
		}
	}
	return boards.ColumnDTO{}
}
