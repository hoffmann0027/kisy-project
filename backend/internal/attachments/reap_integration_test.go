//go:build integration

package attachments_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/attachments"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/quota"
)

// Audit A-07: an upload that is never sent stays in storage forever — and,
// with quotas, keeps eating the uploader's allowance forever too. The reaper
// removes old unlinked uploads, but never one a pending scheduled message is
// still waiting to send.
func TestOldUnlinkedUploadsAreReaped(t *testing.T) {
	svc, pool, ctx := quotaSetup(t)
	user := testdb.SeedUser(t, pool, "uploader", staffLevel)

	upload := func(name string) uuid.UUID {
		t.Helper()
		a, err := svc.Upload(ctx, name, payload(4<<10), user, staffLevel, attachments.Meta{})
		if err != nil {
			t.Fatal(err)
		}
		return a.ID
	}
	age := func(id uuid.UUID, d time.Duration) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE attachments SET created_at = now() - $2::interval WHERE id = $1`,
			id, d.String()); err != nil {
			t.Fatal(err)
		}
	}
	exists := func(id uuid.UUID) bool {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM attachments WHERE id = $1`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n == 1
	}

	abandoned := upload("abandoned.png")
	age(abandoned, 48*time.Hour)
	fresh := upload("fresh.png") // still being composed

	scheduled := upload("scheduled.png")
	age(scheduled, 48*time.Hour)
	if _, err := pool.Exec(ctx, `
		INSERT INTO scheduled_messages (chat_type, chat_id, sender_id, text, attachment_ids, send_at)
		VALUES ('group', gen_random_uuid(), $1, 'later', ARRAY[$2::uuid], now() + interval '30 days')`,
		user, scheduled); err != nil {
		t.Fatal(err)
	}

	sent := upload("sent.png")
	age(sent, 48*time.Hour)
	var msgID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO messages (chat_type, chat_id, sender_id, text)
		VALUES ('group', gen_random_uuid(), $1, 'x') RETURNING id`, user).Scan(&msgID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Link(ctx, pool, []uuid.UUID{sent}, msgID, user); err != nil {
		t.Fatal(err)
	}

	before, err := quota.UserBytes(ctx, pool, user)
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.ReapUnlinked(ctx, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || exists(abandoned) {
		t.Fatalf("an abandoned upload older than a day must be reaped: reaped %d, still there: %v", n, exists(abandoned))
	}
	if !exists(fresh) || !exists(scheduled) || !exists(sent) {
		t.Fatalf("reaped too much: fresh=%v scheduled=%v sent=%v", exists(fresh), exists(scheduled), exists(sent))
	}
	after, err := quota.UserBytes(ctx, pool, user)
	if err != nil {
		t.Fatal(err)
	}
	if after >= before {
		t.Fatalf("reaping must give the quota back: %d bytes before, %d after", before, after)
	}
}
