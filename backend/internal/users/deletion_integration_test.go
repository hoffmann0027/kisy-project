//go:build integration

package users_test

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
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/users"
)

// Deleting your own account (audit E-01). What must be true afterwards: the
// person is gone from everything personal, the conversations other people are
// still having are intact, what they ran has an owner, and the login they used
// can never be taken by someone else.

type deletionFixture struct {
	ctx  context.Context
	pool *pgxpool.Pool
	svc  *users.Service
	me   uuid.UUID
	heir uuid.UUID
	peer uuid.UUID
}

func newDeletionFixture(t *testing.T) deletionFixture {
	t.Helper()
	pool := testdb.New(t)
	svc := users.NewService(pool, users.NewPostgresRepository(),
		audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil))))
	suffix := uuid.NewString()[:8]
	return deletionFixture{
		ctx:  context.Background(),
		pool: pool,
		svc:  svc,
		me:   testdb.SeedUser(t, pool, "leaver_"+suffix, 0),
		heir: testdb.SeedUser(t, pool, "heir_"+suffix, 0),
		peer: testdb.SeedUser(t, pool, "peer_"+suffix, 0),
	}
}

func (f deletionFixture) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(f.ctx, query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// group creates a group owned by owner, with members in the order given.
func (f deletionFixture) group(t *testing.T, owner uuid.UUID, members map[uuid.UUID]string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO groups (name, min_role_level, created_by, kind)
		VALUES ($1, 10, $2, 'group') RETURNING id`, "G"+uuid.NewString()[:8], owner).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	joined := time.Now().Add(-time.Hour)
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO group_members (group_id, user_id, role_in_group, joined_at)
		VALUES ($1, $2, 'owner', $3)`, id, owner, joined); err != nil {
		t.Fatal(err)
	}
	for user, role := range members {
		joined = joined.Add(time.Minute)
		if _, err := f.pool.Exec(f.ctx, `
			INSERT INTO group_members (group_id, user_id, role_in_group, joined_at)
			VALUES ($1, $2, $3, $4)`, id, user, role, joined); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

// chatWith opens a private chat and puts one message from each side in it.
func (f deletionFixture) chatWith(t *testing.T, other uuid.UUID) (chatID, mine, theirs uuid.UUID) {
	t.Helper()
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO private_chats (user_a_id, user_b_id, initiated_by)
		VALUES ($1, $2, $1) RETURNING id`, f.me, other).Scan(&chatID); err != nil {
		t.Fatal(err)
	}
	insert := func(sender uuid.UUID, text string) uuid.UUID {
		var id uuid.UUID
		if err := f.pool.QueryRow(f.ctx, `
			INSERT INTO messages (chat_type, chat_id, sender_id, text)
			VALUES ('private', $1, $2, $3) RETURNING id`, chatID, sender, text).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	return chatID, insert(f.me, "мой секрет"), insert(other, "ответ собеседника")
}

func TestDeletingYourAccountRemovesWhatIsPersonal(t *testing.T) {
	f := newDeletionFixture(t)
	_, mine, theirs := f.chatWith(t, f.peer)
	// Things that belong to this account alone.
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO notes (user_id, text) VALUES ($1, 'заметка')`, f.me); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO device_tokens (user_id, token, platform) VALUES ($1, 'fcm-token', 'android')`, f.me); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO e2ee_devices (id, user_id, name, ed25519_pub)
		VALUES (gen_random_uuid(), $1, 'Phone', decode(repeat('00', 32), 'hex'))`, f.me); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO sessions (user_id, refresh_token_hash, ip_hash, expires_at)
		VALUES ($1, 'hash', 'iphash', now() + interval '30 days')`, f.me); err != nil {
		t.Fatal(err)
	}

	before, err := f.svc.GetByID(f.ctx, f.me)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeleteOwn(f.ctx, f.me, users.ActorMeta{}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	for _, q := range []string{
		`SELECT count(*) FROM notes WHERE user_id = $1`,
		`SELECT count(*) FROM device_tokens WHERE user_id = $1`,
		`SELECT count(*) FROM e2ee_devices WHERE user_id = $1`,
		`SELECT count(*) FROM sessions WHERE user_id = $1`,
	} {
		if n := f.count(t, q, f.me); n != 0 {
			t.Errorf("%s = %d, want 0", q, n)
		}
	}

	// The account is anonymised, not dropped.
	var username, displayName, hash string
	var deletedAt *time.Time
	var active bool
	err = f.pool.QueryRow(f.ctx,
		`SELECT username, display_name, password_hash, is_active, deleted_at FROM users WHERE id = $1`, f.me).
		Scan(&username, &displayName, &hash, &active, &deletedAt)
	if err != nil {
		t.Fatal(err)
	}
	if username == before.Username || displayName != users.DeletedDisplayName || hash != "" || active || deletedAt == nil {
		t.Fatalf("after deletion: username=%q display=%q hash=%q active=%v deletedAt=%v",
			username, displayName, hash, active, deletedAt)
	}

	// The private messages it sent lose their text; the other side keeps theirs.
	var mineText, theirsText *string
	if err := f.pool.QueryRow(f.ctx, `SELECT text FROM messages WHERE id = $1`, mine).Scan(&mineText); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT text FROM messages WHERE id = $1`, theirs).Scan(&theirsText); err != nil {
		t.Fatal(err)
	}
	if mineText != nil {
		t.Errorf("the deleted account's message still reads %q", *mineText)
	}
	if theirsText == nil || *theirsText != "ответ собеседника" {
		t.Errorf("the other side's message was touched: %v", theirsText)
	}
}

// The login is never handed to anyone else: old mentions and links name it.
func TestTheLoginOfADeletedAccountIsNeverReissued(t *testing.T) {
	f := newDeletionFixture(t)
	before, err := f.svc.GetByID(f.ctx, f.me)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeleteOwn(f.ctx, f.me, users.ActorMeta{}); err != nil {
		t.Fatal(err)
	}
	retired, err := users.UsernameRetired(f.ctx, f.pool, before.Username)
	if err != nil {
		t.Fatal(err)
	}
	if !retired {
		t.Fatal("the login was released for anyone to take")
	}
	if retired, _ := users.UsernameRetired(f.ctx, f.pool, "someone_else"); retired {
		t.Fatal("an untouched login is reported as retired")
	}
}

// Two deletions in a row: the second must not trip over the first one's name.
func TestTwoAccountsCanBeDeleted(t *testing.T) {
	f := newDeletionFixture(t)
	if _, err := f.svc.DeleteOwn(f.ctx, f.me, users.ActorMeta{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeleteOwn(f.ctx, f.peer, users.ActorMeta{}); err != nil {
		t.Fatalf("second deletion: %v", err)
	}
}

func TestWhatTheAccountRanPassesOnOrGoes(t *testing.T) {
	f := newDeletionFixture(t)
	withHeir := f.group(t, f.me, map[uuid.UUID]string{f.heir: "editor", f.peer: "member"})
	alone := f.group(t, f.me, nil)

	report, err := f.svc.DeleteOwn(f.ctx, f.me, users.ActorMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if report.GroupsTransferred != 1 || report.GroupsDeleted != 1 {
		t.Fatalf("report = %+v", report)
	}

	var owner uuid.UUID
	var deletedAt *time.Time
	if err := f.pool.QueryRow(f.ctx, `SELECT created_by, deleted_at FROM groups WHERE id = $1`, withHeir).
		Scan(&owner, &deletedAt); err != nil {
		t.Fatal(err)
	}
	if owner != f.heir || deletedAt != nil {
		t.Fatalf("the group with an heir: owner=%v deletedAt=%v", owner, deletedAt)
	}
	if n := f.count(t, `SELECT count(*) FROM group_members WHERE group_id = $1 AND user_id = $2 AND role_in_group = 'owner'`,
		withHeir, f.heir); n != 1 {
		t.Fatalf("the heir is not the owner (%d rows)", n)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT deleted_at FROM groups WHERE id = $1`, alone).Scan(&deletedAt); err != nil {
		t.Fatal(err)
	}
	if deletedAt == nil {
		t.Fatal("a group with nobody left to run it survived its owner")
	}
}

// The account that runs the deployment cannot delete itself: nobody would be
// left to administer what remains.
func TestTheCEOCannotDeleteItself(t *testing.T) {
	f := newDeletionFixture(t)
	ceo := testdb.SeedUser(t, f.pool, "ceo_"+uuid.NewString()[:8], 1)
	if _, err := f.svc.DeleteOwn(f.ctx, ceo, users.ActorMeta{}); !errors.Is(err, users.ErrLastCEO) {
		t.Fatalf("deleting the CEO: %v, want ErrLastCEO", err)
	}
	var deletedAt *time.Time
	if err := f.pool.QueryRow(f.ctx, `SELECT deleted_at FROM users WHERE id = $1`, ceo).Scan(&deletedAt); err != nil {
		t.Fatal(err)
	}
	if deletedAt != nil {
		t.Fatal("the CEO account was deleted anyway")
	}
}
