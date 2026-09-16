//go:build integration

package moderation_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/moderation"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/posts"
)

type recorder struct {
	mu          sync.Mutex
	notices     []moderation.Notice
	feedChanges int
}

func (r *recorder) Notify(_ context.Context, n moderation.Notice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notices = append(r.notices, n)
	return nil
}

type env struct {
	pool   *pgxpool.Pool
	mod    *moderation.Service
	groups *groups.Service
	rec    *recorder
	ceo    moderation.ActorMeta
	owner  uuid.UUID
	editor uuid.UUID
	reader uuid.UUID
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := testdb.New(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := audit.NewPostgresRecorder(log)
	e := &env{pool: pool, rec: &recorder{}}
	e.groups = groups.NewService(pool, groups.NewPostgresRepository(), rec)
	e.mod = moderation.NewService(pool, moderation.NewRepository(), rec, log)
	e.mod.SetNotifier(e.rec)
	e.mod.SetRoleReader(moderation.GroupRoles{Groups: e.groups})
	e.mod.SetFeedChanged(func(context.Context) {
		e.rec.mu.Lock()
		e.rec.feedChanges++
		e.rec.mu.Unlock()
	})
	ceo := testdb.SeedUser(t, pool, "the_ceo", 1)
	e.ceo = moderation.ActorMeta{UserID: ceo, RoleLevel: 1}
	e.owner = testdb.SeedUser(t, pool, "owner", 5)
	e.editor = testdb.SeedUser(t, pool, "editor", 6)
	e.reader = testdb.SeedUser(t, pool, "reader", 7)
	return e
}

func (e *env) community(t *testing.T, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	c, err := e.groups.Create(ctx, groups.CreateInput{Name: name, Kind: groups.KindCommunity, IsPublic: true},
		groups.ActorMeta{UserID: e.owner, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.groups.SetPolicies(ctx, c.ID, groups.PolicyJoinOpen, groups.PolicyPostEditors, groups.ActorMeta{UserID: e.owner, RoleLevel: 5}); err != nil {
		t.Fatal(err)
	}
	for _, u := range []uuid.UUID{e.editor, e.reader} {
		level := map[uuid.UUID]int{e.editor: 6, e.reader: 7}[u]
		if _, err := e.groups.Join(ctx, c.ID, groups.ActorMeta{UserID: u, RoleLevel: level}); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.groups.SetMemberRole(ctx, c.ID, e.editor, groups.RoleEditor, groups.ActorMeta{UserID: e.owner, RoleLevel: 5}); err != nil {
		t.Fatal(err)
	}
	return c.ID
}

func (e *env) warn(t *testing.T, groupID uuid.UUID, reason string) *moderation.Outcome {
	t.Helper()
	out, err := e.mod.Issue(context.Background(), moderation.IssueInput{GroupID: groupID, Kind: moderation.KindWarn, Reason: reason}, e.ceo)
	if err != nil {
		t.Fatalf("warn %q: %v", reason, err)
	}
	return out
}

func (e *env) visibleToReader(t *testing.T, groupID uuid.UUID) bool {
	t.Helper()
	_, err := e.groups.Get(context.Background(), groupID, groups.ActorMeta{UserID: e.reader, RoleLevel: 7})
	if errors.Is(err, groups.ErrNotFound) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

func TestThirdWarningDeletesTheCommunity(t *testing.T) {
	e := setup(t)
	c := e.community(t, "Горный клуб")

	if out := e.warn(t, c, "спам"); out.ActiveWarns != 1 || out.Deleted {
		t.Fatalf("first warning: %+v", out)
	}
	if out := e.warn(t, c, "оскорбления"); out.ActiveWarns != 2 || out.Deleted {
		t.Fatalf("second warning: %+v", out)
	}
	if !e.visibleToReader(t, c) {
		t.Fatal("two warnings must not delete")
	}
	out := e.warn(t, c, "реклама казино")
	if out.ActiveWarns != 3 || !out.Deleted {
		t.Fatalf("third warning must delete: %+v", out)
	}
	if e.visibleToReader(t, c) {
		t.Fatal("a deleted community must be gone for its readers")
	}
	if _, err := e.mod.Issue(context.Background(), moderation.IssueInput{GroupID: c, Kind: moderation.KindWarn, Reason: "ещё"}, e.ceo); !errors.Is(err, moderation.ErrGroupDeleted) {
		t.Fatalf("warning a deleted community: %v", err)
	}

	// The same notice a deletion by hand sends, to the founder and editors —
	// not to plain readers — with the reason in it.
	var deleteNotice *moderation.Notice
	for i := range e.rec.notices {
		if e.rec.notices[i].Payload["action"] == moderation.KindDelete {
			deleteNotice = &e.rec.notices[i]
		}
	}
	if deleteNotice == nil {
		t.Fatal("no deletion notice")
	}
	got := map[uuid.UUID]bool{}
	for _, id := range deleteNotice.Recipients {
		got[id] = true
	}
	if !got[e.owner] || !got[e.editor] || got[e.reader] || len(got) != 2 {
		t.Fatalf("deletion notice recipients = %v", deleteNotice.Recipients)
	}
	if deleteNotice.Payload["reason"] == "" {
		t.Fatal("the reason must reach the recipients")
	}
}

func TestRevokedWarningDoesNotCount(t *testing.T) {
	e := setup(t)
	c := e.community(t, "Шахматы")
	first := e.warn(t, c, "раз")
	e.warn(t, c, "два")

	if _, err := e.mod.Revoke(context.Background(), first.Sanction.ID, "ошибочно", e.ceo); err != nil {
		t.Fatal(err)
	}
	if out := e.warn(t, c, "три"); out.ActiveWarns != 2 || out.Deleted {
		t.Fatalf("with one warning revoked the third must not delete: %+v", out)
	}
	if !e.visibleToReader(t, c) {
		t.Fatal("community deleted despite a revoked warning")
	}
	if _, err := e.mod.Revoke(context.Background(), first.Sanction.ID, "", e.ceo); !errors.Is(err, moderation.ErrNotRevocable) {
		t.Fatalf("revoking twice: %v", err)
	}
}

func TestMutedCommunityIsLeftOutOfTheFeedUntilItExpires(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	c := e.community(t, "Кино")
	other := e.community(t, "Книги")
	repo := posts.NewPostgresRepository()
	postOf := map[uuid.UUID]uuid.UUID{}
	for _, id := range []uuid.UUID{c, other} {
		p := &posts.Post{CommunityID: id, AuthorID: e.owner, Text: "пост"}
		if err := repo.Create(ctx, e.pool, p); err != nil {
			t.Fatal(err)
		}
		postOf[id] = p.ID
	}
	// inPage: the chronological feed page. inRanking: the popularity inputs —
	// checked on their own, since the popular page filters again afterwards
	// and would hide a ranking that still counted a muted community.
	feedHas := func(id uuid.UUID) (inPage, inRanking bool) {
		t.Helper()
		page, err := repo.ListNewest(ctx, e.pool, e.reader, 7, time.Time{}, 50)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range page {
			inPage = inPage || p.CommunityID == id
		}
		inputs, err := repo.ScoreInputs(ctx, e.pool, time.Now().Add(-time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		for _, in := range inputs {
			inRanking = inRanking || in.PostID == postOf[id]
		}
		return inPage, inRanking
	}
	if page, rank := feedHas(c); !page || !rank {
		t.Fatal("precondition: the community's post is in the feed")
	}

	out, err := e.mod.Issue(ctx, moderation.IssueInput{GroupID: c, Kind: moderation.KindMute, Reason: "флуд", Duration: "1d"}, e.ceo)
	if err != nil {
		t.Fatal(err)
	}
	if out.Sanction.ExpiresAt == nil {
		t.Fatal("a one-day mute must expire")
	}
	if page, rank := feedHas(c); page || rank {
		t.Fatalf("muted community still in the feed: page %v, ranking %v", page, rank)
	}
	if page, _ := feedHas(other); !page {
		t.Fatal("a mute must not hide other communities")
	}
	if e.rec.feedChanges == 0 {
		t.Fatal("the popular-feed cache must be recomputed on mute")
	}
	// Muted is not deleted: the wall stays, and its editors may publish.
	if !e.visibleToReader(t, c) {
		t.Fatal("a muted community is still visible on its own page")
	}
	if err := e.groups.EnsureCanPost(ctx, c, groups.ActorMeta{UserID: e.editor, RoleLevel: 6}); err != nil {
		t.Fatalf("editors must still be able to publish while muted: %v", err)
	}

	// Time passes: the mute expires.
	if _, err := e.pool.Exec(ctx,
		`UPDATE group_sanctions SET issued_at = now() - interval '2 days', expires_at = now() - interval '1 minute' WHERE id = $1`,
		out.Sanction.ID); err != nil {
		t.Fatal(err)
	}
	if page, rank := feedHas(c); !page || !rank {
		t.Fatalf("expired mute still hides the community: page %v, ranking %v", page, rank)
	}
}

func TestRevokingAMuteBringsTheCommunityBackAndRecomputesTheFeed(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	c := e.community(t, "Музыка")
	out, err := e.mod.Issue(ctx, moderation.IssueInput{GroupID: c, Kind: moderation.KindMute, Reason: "пауза", Duration: "forever"}, e.ceo)
	if err != nil || out.Sanction.ExpiresAt != nil {
		t.Fatalf("indefinite mute: %v %+v", err, out)
	}
	before := e.rec.feedChanges
	if _, err := e.mod.Revoke(ctx, out.Sanction.ID, "", e.ceo); err != nil {
		t.Fatal(err)
	}
	if e.rec.feedChanges == before {
		t.Fatal("lifting a mute must recompute the popular feed")
	}
	active, err := e.mod.ActiveFor(ctx, c, e.ceo)
	if err != nil || active.Mute != nil {
		t.Fatalf("mute still live after revoke: %v %+v", err, active)
	}
}

func TestRestoreBringsEverythingBackAndLeavesOneWarningToGo(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	c := e.community(t, "Фото")
	repo := posts.NewPostgresRepository()
	if err := repo.Create(ctx, e.pool, &posts.Post{CommunityID: c, AuthorID: e.owner, Text: "пост"}); err != nil {
		t.Fatal(err)
	}
	e.warn(t, c, "a")
	e.warn(t, c, "b")
	e.warn(t, c, "c")

	deleted, err := e.mod.ListDeleted(ctx)
	if err != nil || len(deleted) != 1 || !deleted[0].PurgeAt.Equal(deleted[0].DeletedAt.Add(moderation.RestoreWindow)) {
		t.Fatalf("deleted list: %v %+v", err, deleted)
	}

	e.rec.notices = nil
	if err := e.mod.Restore(ctx, c, e.ceo); err != nil {
		t.Fatal(err)
	}
	if !e.visibleToReader(t, c) {
		t.Fatal("restored community must be visible again")
	}
	if member, _ := e.groups.IsMember(ctx, c, e.reader); !member {
		t.Fatal("members must come back with it")
	}
	wall, _ := repo.ListByCommunity(ctx, e.pool, c, time.Time{}, 10)
	if len(wall) != 1 {
		t.Fatal("posts must come back with it")
	}
	if len(e.rec.notices) != 1 || e.rec.notices[0].Recipients[0] != e.owner || e.rec.notices[0].Payload["action"] != "restore" {
		t.Fatalf("the founder must be told about the restore: %+v", e.rec.notices)
	}
	if out := e.warn(t, c, "d"); !out.Deleted {
		t.Fatalf("after a restore the next warning must delete again: %+v", out)
	}
	if err := e.mod.Restore(ctx, c, e.ceo); err != nil {
		t.Fatal(err)
	}
	if err := e.mod.Restore(ctx, c, e.ceo); !errors.Is(err, moderation.ErrNotDeleted) {
		t.Fatalf("restoring a live community: %v", err)
	}
}

func TestRestoreWindowAndPurge(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	c := e.community(t, "Старый")
	if _, err := e.mod.Issue(ctx, moderation.IssueInput{GroupID: c, Kind: moderation.KindDelete, Reason: "закрыть"}, e.ceo); err != nil {
		t.Fatal(err)
	}
	// Within the window nothing is purged.
	if n, err := e.mod.Purge(ctx, time.Now()); err != nil || n != 0 {
		t.Fatalf("purge within the window: %d %v", n, err)
	}
	// 31 days later.
	later := time.Now().Add(moderation.RestoreWindow + 24*time.Hour)
	e.mod.SetClock(func() time.Time { return later })
	if err := e.mod.Restore(ctx, c, e.ceo); !errors.Is(err, moderation.ErrRestoreExpired) {
		t.Fatalf("restore after the window: %v", err)
	}
	if n, err := e.mod.Purge(ctx, later); err != nil || n != 1 {
		t.Fatalf("purge after the window: %d %v", n, err)
	}
	var exists bool
	_ = e.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM groups WHERE id = $1)`, c).Scan(&exists)
	if exists {
		t.Fatal("purged group still in the database")
	}
	var audited int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action = 'group.purged' AND target_id = $1`, c).Scan(&audited)
	if audited != 1 {
		t.Fatalf("purge must be audited, got %d rows", audited)
	}
}

func TestReasonIsRequiredAndOnlyTheCEOMayAct(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	c := e.community(t, "Правила")
	if _, err := e.mod.Issue(ctx, moderation.IssueInput{GroupID: c, Kind: moderation.KindWarn, Reason: "   "}, e.ceo); !errors.Is(err, moderation.ErrReasonRequired) {
		t.Fatalf("blank reason: %v", err)
	}
	notCEO := moderation.ActorMeta{UserID: e.owner, RoleLevel: 2}
	if _, err := e.mod.Issue(ctx, moderation.IssueInput{GroupID: c, Kind: moderation.KindWarn, Reason: "x"}, notCEO); !errors.Is(err, moderation.ErrForbidden) {
		t.Fatalf("non-CEO issue: %v", err)
	}
	if err := e.mod.Restore(ctx, c, notCEO); !errors.Is(err, moderation.ErrForbidden) {
		t.Fatalf("non-CEO restore: %v", err)
	}
	out := e.warn(t, c, "x")
	if _, err := e.mod.Revoke(ctx, out.Sanction.ID, "", notCEO); !errors.Is(err, moderation.ErrForbidden) {
		t.Fatalf("non-CEO revoke: %v", err)
	}

	// The banner: editors of the community see it, plain readers do not.
	if _, err := e.mod.ActiveFor(ctx, c, moderation.ActorMeta{UserID: e.editor, RoleLevel: 6}); err != nil {
		t.Fatalf("editor banner: %v", err)
	}
	if _, err := e.mod.ActiveFor(ctx, c, moderation.ActorMeta{UserID: e.reader, RoleLevel: 7}); !errors.Is(err, moderation.ErrForbidden) {
		t.Fatalf("reader banner: %v", err)
	}
}
