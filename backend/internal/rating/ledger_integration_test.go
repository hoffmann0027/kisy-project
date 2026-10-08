//go:build integration

package rating_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/rating"
)

// The dashboard shows income and expense apart, a monthly series of both, a
// feed of the latest entries and when a project last moved. The board and the
// analytics used to carry profit alone — nothing a dashboard could be built
// from.

func TestBoardAndAnalyticsCarryTheLedger(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	svc := rating.NewService(pool, rating.NewPostgresRepository())
	ceo := rating.Actor{UserID: testdb.SeedUser(t, pool, "ceo_"+uuid.NewString()[:8], 1), RoleLevel: 1}

	id, err := svc.CreateProject(ctx, rating.CreateProjectInput{Title: "Ledger", Difficulty: "medium", MinLevel: 10}, ceo)
	if err != nil {
		t.Fatal(err)
	}
	// Two entries this month: 1 000 € in, 350 € out. The project row itself
	// is never updated, so the ledger is what dates the last change.
	for _, in := range []rating.FinanceInput{
		{IncomeKopecks: 100_000},
		{ExpenseKopecks: 35_000, Note: strPtr("hosting")},
	} {
		if err := svc.AddFinance(ctx, id, in, ceo); err != nil {
			t.Fatal(err)
		}
	}

	board, err := svc.Board(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	var p *rating.ProjectDTO
	for i := range board.Projects {
		if board.Projects[i].ID == id {
			p = &board.Projects[i]
		}
	}
	if p == nil {
		t.Fatal("the project is not on the board")
	}
	if p.TotalIncomeKopecks != 100_000 || p.TotalExpenseKopecks != 35_000 || p.TotalProfitKopecks != 65_000 {
		t.Fatalf("totals = %d/%d/%d, want 100000/35000/65000", p.TotalIncomeKopecks, p.TotalExpenseKopecks, p.TotalProfitKopecks)
	}
	if !p.UpdatedAt.After(p.CreatedAt) {
		t.Fatalf("updatedAt %v is not after createdAt %v although money was recorded since", p.UpdatedAt, p.CreatedAt)
	}
	if p.CompletedAt != nil {
		t.Fatal("an active project has a completion date")
	}

	a, err := svc.Analytics(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	month := time.Now().UTC().Format("2006-01")
	var mine *rating.MonthlyProfit
	for i := range a.Monthly {
		if a.Monthly[i].Month == month {
			mine = &a.Monthly[i]
		}
	}
	if mine == nil {
		t.Fatalf("no row for %s in %v", month, a.Monthly)
	}
	if mine.IncomeKopecks < 100_000 || mine.ExpenseKopecks < 35_000 || mine.ProfitKopecks != mine.IncomeKopecks-mine.ExpenseKopecks {
		t.Fatalf("this month = %+v, want at least 100000 in and 35000 out, profit their difference", *mine)
	}
	var share *rating.ProjectProfit
	for i := range a.PerProject {
		if a.PerProject[i].ProjectID == id {
			share = &a.PerProject[i]
		}
	}
	if share == nil || share.IncomeKopecks != 100_000 || share.ExpenseKopecks != 35_000 {
		t.Fatalf("per-project = %+v, want income 100000 and expense 35000", share)
	}

	// The feed: newest first, with the project and the note.
	if len(a.Recent) < 2 {
		t.Fatalf("recent = %d entries, want at least the two just recorded", len(a.Recent))
	}
	if len(a.Recent) > rating.RecentLedgerLimit {
		t.Fatalf("recent carries %d entries, more than the %d a feed shows", len(a.Recent), rating.RecentLedgerLimit)
	}
	first := a.Recent[0]
	if first.ProjectID != id || first.ExpenseKopecks != 35_000 || first.Note == nil || *first.Note != "hosting" {
		t.Fatalf("newest entry = %+v, want the 350 € hosting expense of the project", first)
	}
	if first.AuthorName == "" {
		t.Fatal("the entry does not say who recorded it")
	}
	if a.Recent[1].IncomeKopecks != 100_000 {
		t.Fatalf("second entry = %+v, want the 1000 € income", a.Recent[1])
	}
}

func strPtr(s string) *string { return &s }
