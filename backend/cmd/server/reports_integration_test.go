//go:build integration

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/posts"
	"kisy-backend/internal/reports"
)

// Reporting content, and what the queue does with it (audit E-02): five
// different people take a post out of the feed until the CEO decides, one
// person takes nothing out, the author keeps seeing their own post, and a
// private message is reported without being decrypted.

type reportFixture struct {
	ctx     context.Context
	pool    *pgxpool.Pool
	svc     *reports.Service
	posts   *posts.Service
	groups  *groups.Service
	author  uuid.UUID
	post    posts.DTO
	readers []uuid.UUID
}

func newReportFixture(t *testing.T) reportFixture {
	t.Helper()
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(quiet())
	suffix := uuid.NewString()[:8]

	f := reportFixture{ctx: ctx, pool: pool}
	f.svc = reports.NewService(pool, rec)
	wireReports(f.svc, pool)
	f.groups = groups.NewService(pool, groups.NewPostgresRepository(), rec)
	f.posts = posts.NewService(pool, posts.NewPostgresRepository(), postsCommunities{groups: f.groups}, rec)

	f.author = testdb.SeedUser(t, pool, "author_"+suffix, 5)
	actor := groups.ActorMeta{UserID: f.author, RoleLevel: 5}
	g, err := f.groups.Create(ctx, groups.CreateInput{Name: "Wall " + suffix, Kind: groups.KindCommunity, IsPublic: true}, actor)
	if err != nil {
		t.Fatal(err)
	}
	dto, err := f.posts.Create(ctx, posts.CreateInput{CommunityID: g.ID, Text: "спорный пост"}, posts.ActorMeta{UserID: f.author, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	f.post = *dto
	for i := 0; i < reports.AutoHideThreshold+1; i++ {
		f.readers = append(f.readers, testdb.SeedUser(t, pool, "reader"+string(rune('a'+i))+"_"+suffix, 5))
	}
	return f
}

func (f reportFixture) report(t *testing.T, reporter uuid.UUID, kind string, id uuid.UUID) {
	t.Helper()
	if _, err := f.svc.Create(f.ctx, reports.ActorMeta{UserID: reporter}, reports.Input{
		TargetKind: kind, TargetID: id, Reason: "spam", Comment: "надоело",
	}); err != nil {
		t.Fatal(err)
	}
}

func (f reportFixture) inFeedOf(t *testing.T, viewer uuid.UUID) bool {
	t.Helper()
	page, err := f.posts.Feed(f.ctx, "new", "", 50, posts.ActorMeta{UserID: viewer, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range page.Posts {
		if p.ID == f.post.ID {
			return true
		}
	}
	return false
}

func TestFiveReportsHideAPostButOneDoesNot(t *testing.T) {
	f := newReportFixture(t)

	f.report(t, f.readers[0], reports.TargetPost, f.post.ID)
	if !f.inFeedOf(t, f.readers[1]) {
		t.Fatal("one report hid the post; one person must not be able to")
	}

	for i := 1; i < reports.AutoHideThreshold; i++ {
		f.report(t, f.readers[i], reports.TargetPost, f.post.ID)
	}
	if f.inFeedOf(t, f.readers[reports.AutoHideThreshold]) {
		t.Fatalf("%d reports did not take the post out of the feed", reports.AutoHideThreshold)
	}
	// The author still sees their own post: hiding it from them as well would
	// let a small group silence someone without them ever knowing why.
	if !f.inFeedOf(t, f.author) {
		t.Fatal("the author lost sight of their own post")
	}

	// Once the CEO has judged them, the post comes back.
	queue, err := f.svc.List(f.ctx, reports.StatusOpen, 0)
	if err != nil {
		t.Fatal(err)
	}
	ceo := testdb.SeedUser(t, f.pool, "ceo_"+uuid.NewString()[:8], 1)
	for _, rep := range queue {
		if err := f.svc.Resolve(f.ctx, reports.ActorMeta{UserID: ceo}, rep.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	if !f.inFeedOf(t, f.readers[reports.AutoHideThreshold]) {
		t.Fatal("the post stayed hidden after every report was rejected")
	}
}

func TestTheQueueCountsPeopleNotPresses(t *testing.T) {
	f := newReportFixture(t)
	f.report(t, f.readers[0], reports.TargetPost, f.post.ID)
	f.report(t, f.readers[0], reports.TargetPost, f.post.ID)
	f.report(t, f.readers[1], reports.TargetPost, f.post.ID)

	queue, err := f.svc.List(f.ctx, reports.StatusOpen, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(queue) != 2 {
		t.Fatalf("queue has %d entries, want 2 (one per person)", len(queue))
	}
	for _, rep := range queue {
		if rep.SameTarget != 2 {
			t.Fatalf("sameTarget = %d, want 2", rep.SameTarget)
		}
		if rep.TargetOwner == nil || *rep.TargetOwner != f.author {
			t.Fatalf("targetOwner = %v, want the author", rep.TargetOwner)
		}
		if !rep.Readable || rep.Content == nil || *rep.Content != "спорный пост" {
			t.Fatalf("a post's text must be in the queue: %+v", rep)
		}
	}
}

// A private message is reported without being decrypted: the queue says the
// content is not readable rather than showing an empty text.
func TestAPrivateMessageIsReportedWithoutBeingRead(t *testing.T) {
	f := newReportFixture(t)
	var chatID, messageID uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO private_chats (user_a_id, user_b_id, initiated_by)
		VALUES ($1, $2, $1) RETURNING id`, f.author, f.readers[0]).Scan(&chatID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO messages (chat_type, chat_id, sender_id, ciphertext, alg, epoch)
		VALUES ('private', $1, $2, decode('00', 'hex'), 1, 1) RETURNING id`, chatID, f.author).Scan(&messageID); err != nil {
		t.Fatal(err)
	}

	f.report(t, f.readers[0], reports.TargetMessage, messageID)

	queue, err := f.svc.List(f.ctx, reports.StatusOpen, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(queue) != 1 {
		t.Fatalf("queue = %d entries", len(queue))
	}
	if queue[0].Readable || queue[0].Content != nil {
		t.Fatalf("a private message must not be readable in the queue: %+v", queue[0])
	}
	if queue[0].TargetOwner == nil || *queue[0].TargetOwner != f.author {
		t.Fatal("the queue must still know whose message it was")
	}
}

func TestYouCannotReportYourselfAndNonsenseIsRefused(t *testing.T) {
	f := newReportFixture(t)
	_, err := f.svc.Create(f.ctx, reports.ActorMeta{UserID: f.author}, reports.Input{
		TargetKind: reports.TargetPost, TargetID: f.post.ID, Reason: "spam",
	})
	if !errors.Is(err, reports.ErrSelf) {
		t.Fatalf("reporting your own post: %v", err)
	}
	_, err = f.svc.Create(f.ctx, reports.ActorMeta{UserID: f.readers[0]}, reports.Input{
		TargetKind: "nonsense", TargetID: f.post.ID, Reason: "spam",
	})
	if !errors.Is(err, reports.ErrBadTarget) {
		t.Fatalf("unknown target: %v", err)
	}
	_, err = f.svc.Create(f.ctx, reports.ActorMeta{UserID: f.readers[0]}, reports.Input{
		TargetKind: reports.TargetPost, TargetID: f.post.ID, Reason: "because",
	})
	if !errors.Is(err, reports.ErrBadReason) {
		t.Fatalf("unknown reason: %v", err)
	}
}

func TestResolvingIsRecordedAndHappensOnce(t *testing.T) {
	f := newReportFixture(t)
	f.report(t, f.readers[0], reports.TargetPost, f.post.ID)
	queue, err := f.svc.List(f.ctx, reports.StatusOpen, 0)
	if err != nil || len(queue) != 1 {
		t.Fatalf("queue: %v %d", err, len(queue))
	}
	ceo := testdb.SeedUser(t, f.pool, "ceo_"+uuid.NewString()[:8], 1)

	if err := f.svc.Resolve(f.ctx, reports.ActorMeta{UserID: ceo}, queue[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Resolve(f.ctx, reports.ActorMeta{UserID: ceo}, queue[0].ID, false); !errors.Is(err, reports.ErrNotFound) {
		t.Fatalf("resolving twice: %v, want ErrNotFound", err)
	}

	counts, err := f.svc.Summary(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Open != 0 || counts.Resolved != 1 {
		t.Fatalf("counts = %+v", counts)
	}
	var audited int
	if err := f.pool.QueryRow(f.ctx,
		`SELECT count(*) FROM audit_logs WHERE action LIKE 'report.%'`).Scan(&audited); err != nil {
		t.Fatal(err)
	}
	if audited != 2 { // the report and its resolution
		t.Fatalf("audit entries = %d, want 2", audited)
	}
}
