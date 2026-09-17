//go:build integration

package groups_test

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

// Audit A-05: the recipients of a group's real-time events (message.created,
// typing, reactions, board…), its push notifications and its read counts were
// every row in group_members. A member whose clearance no longer cleared the
// group, a member of a group deleted by moderation, and a deactivated account
// kept receiving new messages over the socket while REST answered 404.

type recipientsFixture struct {
	ctx       context.Context
	svc       *groups.Service
	exec      func(sql string, args ...any)
	group     uuid.UUID
	low, high uuid.UUID
}

func newRecipientsFixture(t *testing.T) recipientsFixture {
	t.Helper()
	pool := testdb.New(t)
	ctx := context.Background()
	f := recipientsFixture{
		ctx:  ctx,
		svc:  groups.NewService(pool, groups.NewPostgresRepository(), audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))),
		low:  testdb.SeedUser(t, pool, "rcpt_low", 10),
		high: testdb.SeedUser(t, pool, "rcpt_high", 3),
	}
	f.exec = func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO groups (name, min_role_level, created_by) VALUES ('Recipients', 10, $1) RETURNING id`, f.high).Scan(&f.group); err != nil {
		t.Fatal(err)
	}
	f.exec(`INSERT INTO group_members (group_id, user_id) VALUES ($1, $2), ($1, $3)`, f.group, f.low, f.high)
	return f
}

func (f recipientsFixture) recipients(t *testing.T) []uuid.UUID {
	t.Helper()
	ids, err := f.svc.MemberIDs(f.ctx, f.group)
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestGroupRecipientsAreMembersWhoCanSeeIt(t *testing.T) {
	f := newRecipientsFixture(t)
	got := f.recipients(t)
	if !slices.Contains(got, f.low) || !slices.Contains(got, f.high) || len(got) != 2 {
		t.Fatalf("both members clear the group: %v", got)
	}
}

func TestGroupRecipientsDropMemberBelowRaisedThreshold(t *testing.T) {
	f := newRecipientsFixture(t)
	f.exec(`UPDATE groups SET min_role_level = 5 WHERE id = $1`, f.group)
	got := f.recipients(t)
	if slices.Contains(got, f.low) {
		t.Fatalf("a level-10 member still receives a level-5 group's events: %v", got)
	}
	if !slices.Contains(got, f.high) {
		t.Fatalf("the level-3 member must keep receiving: %v", got)
	}

	// The content-free "group changed" signal still reaches them, so their
	// client refetches and drops the group.
	rows, err := f.svc.MemberRowIDs(f.ctx, f.group)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(rows, f.low) {
		t.Fatalf("the member who lost access must still be told the group changed: %v", rows)
	}
}

func TestGroupRecipientsEmptyForDeletedOrArchivedGroup(t *testing.T) {
	f := newRecipientsFixture(t)
	f.exec(`UPDATE groups SET deleted_at = now() WHERE id = $1`, f.group)
	if got := f.recipients(t); len(got) != 0 {
		t.Fatalf("a deleted group still has recipients: %v", got)
	}
	f.exec(`UPDATE groups SET deleted_at = NULL, is_archived = true WHERE id = $1`, f.group)
	if got := f.recipients(t); len(got) != 0 {
		t.Fatalf("an archived group still has recipients: %v", got)
	}
}

func TestGroupRecipientsDropDeactivatedAccount(t *testing.T) {
	f := newRecipientsFixture(t)
	f.exec(`UPDATE users SET is_active = false WHERE id = $1`, f.low)
	if got := f.recipients(t); slices.Contains(got, f.low) {
		t.Fatalf("a deactivated account still receives the group's events: %v", got)
	}
}
