//go:build integration

package search_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/search"
)

// Audit A-04: search scoped group messages by membership alone. A member
// kept finding a group's messages after the group was deleted by moderation,
// archived, or raised above their clearance — including messages written
// after they lost access.

type searchFixture struct {
	ctx   context.Context
	svc   *search.Service
	exec  func(sql string, args ...any)
	group uuid.UUID
	low   uuid.UUID // level 10: clears the group only while its threshold is 10
	high  uuid.UUID // level 3
}

func newSearchFixture(t *testing.T) searchFixture {
	t.Helper()
	pool := testdb.New(t)
	ctx := context.Background()
	f := searchFixture{
		ctx:  ctx,
		svc:  search.NewService(pool, slog.New(slog.NewTextHandler(io.Discard, nil))),
		low:  testdb.SeedUser(t, pool, "search_low", 10),
		high: testdb.SeedUser(t, pool, "search_high", 3),
	}
	f.exec = func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO groups (name, min_role_level, created_by) VALUES ('Search group', 10, $1) RETURNING id`,
		f.high).Scan(&f.group); err != nil {
		t.Fatal(err)
	}
	f.exec(`INSERT INTO group_members (group_id, user_id) VALUES ($1, $2), ($1, $3)`, f.group, f.low, f.high)

	var msg uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO messages (chat_type, chat_id, sender_id, text) VALUES ('group', $1, $2, 'квартальный бюджет') RETURNING id`,
		f.group, f.high).Scan(&msg); err != nil {
		t.Fatal(err)
	}
	f.svc.IndexMessage(ctx, msg, "квартальный бюджет")
	return f
}

func (f searchFixture) hits(t *testing.T, who uuid.UUID) int {
	t.Helper()
	res, err := f.svc.Search(f.ctx, who, "бюджет", 25)
	if err != nil {
		t.Fatal(err)
	}
	return len(res)
}

func TestSearchFindsGroupMessagesMembersCanSee(t *testing.T) {
	f := newSearchFixture(t)
	if f.hits(t, f.low) != 1 || f.hits(t, f.high) != 1 {
		t.Fatalf("both members clear the group and must find its message: low=%d high=%d", f.hits(t, f.low), f.hits(t, f.high))
	}
}

func TestSearchHidesGroupRaisedAboveMemberClearance(t *testing.T) {
	f := newSearchFixture(t)
	f.exec(`UPDATE groups SET min_role_level = 5 WHERE id = $1`, f.group)
	if n := f.hits(t, f.low); n != 0 {
		t.Fatalf("a level-10 member of a level-5 group found %d messages", n)
	}
	if n := f.hits(t, f.high); n != 1 {
		t.Fatalf("a level-3 member still clears the group, found %d", n)
	}
}

func TestSearchHidesDeletedGroup(t *testing.T) {
	f := newSearchFixture(t)
	f.exec(`UPDATE groups SET deleted_at = now() WHERE id = $1`, f.group)
	if n := f.hits(t, f.high); n != 0 {
		t.Fatalf("a deleted group's messages are still searchable (%d)", n)
	}
}

func TestSearchHidesArchivedGroup(t *testing.T) {
	f := newSearchFixture(t)
	f.exec(`UPDATE groups SET is_archived = true WHERE id = $1`, f.group)
	if n := f.hits(t, f.high); n != 0 {
		t.Fatalf("an archived group's messages are still searchable (%d)", n)
	}
}
