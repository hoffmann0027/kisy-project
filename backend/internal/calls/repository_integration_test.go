//go:build integration

package calls_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/calls"
	"kisy-backend/internal/platform/testdb"
)

// TestCallLogRepository validates the call_logs SQL end to end against a real
// database: create, finalize, and list from both participants' perspectives.
func TestCallLogRepository(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	repo := calls.NewPostgresRepository()

	caller := testdb.SeedUser(t, pool, "caller", 3)
	callee := testdb.SeedUser(t, pool, "callee", 5)
	chatID := uuid.New()
	id := uuid.New()

	started := time.Now().UTC()
	if err := repo.Create(ctx, pool, calls.CallLog{
		ID: id, CallerID: caller, CalleeID: callee, ChatID: chatID,
		Status: calls.StatusMissed, StartedAt: started,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	answered := started.Add(3 * time.Second)
	ended := answered.Add(42 * time.Second)
	if err := repo.Finalize(ctx, pool, id, calls.StatusCompleted, &answered, &ended, 42); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	rows, err := repo.ListForUser(ctx, pool, caller, 50, 0)
	if err != nil {
		t.Fatalf("list caller: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("caller history len = %d, want 1", len(rows))
	}
	r := rows[0]
	if r.Status != calls.StatusCompleted || r.DurationSeconds != 42 {
		t.Fatalf("row = %+v, want completed/42s", r)
	}
	if r.CalleeName != "callee" || r.CallerName != "caller" {
		t.Fatalf("joined names = %q/%q", r.CallerName, r.CalleeName)
	}
	if r.AnsweredAt == nil || r.EndedAt == nil {
		t.Fatal("expected answered_at and ended_at to be set")
	}

	// The callee sees the same call in their history.
	calleeRows, err := repo.ListForUser(ctx, pool, callee, 50, 0)
	if err != nil {
		t.Fatalf("list callee: %v", err)
	}
	if len(calleeRows) != 1 || calleeRows[0].ID != id {
		t.Fatalf("callee history = %+v, want the same call", calleeRows)
	}
}
