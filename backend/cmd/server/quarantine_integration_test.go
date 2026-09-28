//go:build integration

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/attachments"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/config"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/notes"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/posts"
	"kisy-backend/internal/quarantine"
	"kisy-backend/internal/users"
)

// The new-account quarantine through the real services, wired exactly as the
// server wires them (wireQuarantine). An account that registered an hour ago
// may write to people, but may not publish, may not found a community, may not
// upload big files, may only start a few conversations a day — and its
// reactions do not lift anything in the feed.

type heldFixture struct {
	ctx     context.Context
	pool    *pgxpool.Pool
	posts   *posts.Service
	groups  *groups.Service
	files   *attachments.Service
	notes   *notes.Service
	chats   *chats.Service
	checker *quarantine.Checker
	// fresh registered an hour ago; mature two days ago; invited came through
	// an invitation and is never held.
	fresh, mature, invited uuid.UUID
}

func newHeldFixture(t *testing.T, policy config.QuarantineConfig) heldFixture {
	t.Helper()
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(quiet())
	suffix := uuid.NewString()[:8]

	f := heldFixture{ctx: ctx, pool: pool}
	f.fresh = seedAged(t, pool, "held_"+suffix, 0, time.Hour)
	f.mature = seedAged(t, pool, "old_"+suffix, 0, 48*time.Hour)
	f.invited = seedAged(t, pool, "inv_"+suffix, 5, time.Minute)

	f.groups = groups.NewService(pool, groups.NewPostgresRepository(), rec)
	f.posts = posts.NewService(pool, posts.NewPostgresRepository(), postsCommunities{groups: f.groups}, rec)
	f.files = attachments.NewService(pool, attachments.NewPostgresRepository(), attachments.Limits{
		MaxBytesLeadership: 50 << 20, MaxBytesStaff: 50 << 20, MaxBytesBasic: 50 << 20,
		ChunkBytes: 1 << 20, SessionTTL: time.Hour,
	})
	f.notes = notes.NewService(pool, notes.NewPostgresRepository())
	f.chats = chats.NewService(pool, chats.NewPostgresRepository(), func(_ context.Context, id uuid.UUID) (int, bool) {
		var level *int
		if err := pool.QueryRow(ctx, `SELECT role_id FROM users WHERE id = $1`, id).Scan(&level); err != nil {
			return 0, false
		}
		if level == nil {
			return 0, true
		}
		return *level, true
	})

	f.checker = wireQuarantine(policy, pool, users.NewPostgresRepository(), heldBack{
		posts: f.posts, groups: f.groups, attachments: f.files, notes: f.notes, chats: f.chats,
	})
	return f
}

// seedAged seeds an account and backdates its registration.
func seedAged(t *testing.T, pool *pgxpool.Pool, name string, level int, age time.Duration) uuid.UUID {
	t.Helper()
	id := testdb.SeedUser(t, pool, name, level)
	if _, err := pool.Exec(context.Background(),
		`UPDATE users SET created_at = now() - $2::interval WHERE id = $1`,
		id, age.String()); err != nil {
		t.Fatal(err)
	}
	return id
}

func defaultPolicy() config.QuarantineConfig {
	return config.QuarantineConfig{Hours: 24, NewChatsPerDay: 3, MaxUploadBytes: 2 << 20}
}

// community founds a public community anyone may join and post in, so the
// tests below fail on the quarantine and never on membership.
func (f heldFixture) community(t *testing.T, founder uuid.UUID) uuid.UUID {
	t.Helper()
	actor := groups.ActorMeta{UserID: founder, RoleLevel: 5}
	g, err := f.groups.Create(f.ctx, groups.CreateInput{Name: "Wall " + uuid.NewString()[:6], Kind: groups.KindCommunity, IsPublic: true}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.groups.SetPolicies(f.ctx, g.ID, groups.PolicyJoinOpen, groups.PolicyPostAll, actor); err != nil {
		t.Fatal(err)
	}
	return g.ID
}

// join makes the account a member of a community that anyone may join.
func (f heldFixture) join(t *testing.T, community, user uuid.UUID) {
	t.Helper()
	if _, err := f.groups.Join(f.ctx, community, groups.ActorMeta{UserID: user}); err != nil {
		t.Fatal(err)
	}
}

func TestHeldAccountCannotPublishOrFoundACommunity(t *testing.T) {
	f := newHeldFixture(t, defaultPolicy())
	wall := f.community(t, f.invited)
	f.join(t, wall, f.fresh)

	_, err := f.posts.Create(f.ctx, posts.CreateInput{CommunityID: wall, Text: "buy cheap followers"}, posts.ActorMeta{UserID: f.fresh})
	if !errors.Is(err, quarantine.ErrPosts) {
		t.Fatalf("posting from a fresh account: %v, want ErrPosts", err)
	}
	_, err = f.groups.Create(f.ctx, groups.CreateInput{Name: "Spam " + uuid.NewString()[:6], Kind: groups.KindCommunity, IsPublic: true},
		groups.ActorMeta{UserID: f.fresh})
	if !errors.Is(err, quarantine.ErrCommunities) {
		t.Fatalf("founding a community: %v, want ErrCommunities", err)
	}

	// An ordinary group is not a community and stays open to everyone.
	if _, err := f.groups.Create(f.ctx, groups.CreateInput{Name: "Talk " + uuid.NewString()[:6], Kind: groups.KindGroup},
		groups.ActorMeta{UserID: f.fresh}); err != nil {
		t.Fatalf("an ordinary group: %v", err)
	}
}

func TestMatureAndInvitedAccountsPublishFreely(t *testing.T) {
	f := newHeldFixture(t, defaultPolicy())
	wall := f.community(t, f.invited)
	f.join(t, wall, f.mature)
	for name, author := range map[string]uuid.UUID{"invited": f.invited, "two days old": f.mature} {
		if _, err := f.posts.Create(f.ctx, posts.CreateInput{CommunityID: wall, Text: "hello"}, posts.ActorMeta{UserID: author, RoleLevel: 5}); err != nil {
			t.Fatalf("%s: publishing refused: %v", name, err)
		}
	}
}

func TestHeldAccountCannotUploadBigFiles(t *testing.T) {
	f := newHeldFixture(t, defaultPolicy())
	big := make([]byte, (2<<20)+1)
	small := []byte("small file, perfectly fine")

	if _, err := f.files.Upload(f.ctx, "big.bin", big, f.fresh, 0, attachments.Meta{}); !errors.Is(err, quarantine.ErrUpload) {
		t.Fatalf("3 MiB attachment: %v, want ErrUpload", err)
	}
	if _, err := f.files.InitUpload(f.ctx, f.fresh, 0, "big.bin", 10<<20, attachments.Meta{}); !errors.Is(err, quarantine.ErrUpload) {
		t.Fatalf("chunked upload of 10 MiB: %v, want ErrUpload", err)
	}
	if _, err := f.notes.CreateFile(f.ctx, f.fresh, "big.bin", "", big); !errors.Is(err, quarantine.ErrUpload) {
		t.Fatalf("note file of 2 MiB+1: %v, want ErrUpload", err)
	}
	// Small files keep working throughout the hold.
	if _, err := f.files.Upload(f.ctx, "small.txt", small, f.fresh, 0, attachments.Meta{}); err != nil {
		t.Fatalf("small attachment: %v", err)
	}
	// A mature account is not capped.
	if _, err := f.files.Upload(f.ctx, "big.bin", big, f.mature, 0, attachments.Meta{}); err != nil {
		t.Fatalf("mature account, 2 MiB+1: %v", err)
	}
}

func TestHeldAccountGetsOnlyAFewNewConversationsADay(t *testing.T) {
	f := newHeldFixture(t, defaultPolicy()) // 3 per day
	targets := make([]uuid.UUID, 0, 4)
	for i := 0; i < 4; i++ {
		targets = append(targets, seedAged(t, f.pool, "target_"+uuid.NewString()[:8], 0, time.Hour))
	}
	for i := 0; i < 3; i++ {
		if _, err := f.chats.OpenPrivateChat(f.ctx, targets[i], chats.ActorMeta{UserID: f.fresh}); err != nil {
			t.Fatalf("conversation %d: %v", i+1, err)
		}
	}
	_, err := f.chats.OpenPrivateChat(f.ctx, targets[3], chats.ActorMeta{UserID: f.fresh})
	if !errors.Is(err, quarantine.ErrNewChats) {
		t.Fatalf("the fourth conversation: %v, want ErrNewChats", err)
	}
	// Reopening one that exists is free, and answering is never affected.
	if _, err := f.chats.OpenPrivateChat(f.ctx, targets[0], chats.ActorMeta{UserID: f.fresh}); err != nil {
		t.Fatalf("reopening an existing conversation: %v", err)
	}
	// A mature account is not capped.
	for i := 0; i < 4; i++ {
		if _, err := f.chats.OpenPrivateChat(f.ctx, targets[i], chats.ActorMeta{UserID: f.mature}); err != nil {
			t.Fatalf("mature account, conversation %d: %v", i+1, err)
		}
	}
}

// Reactions from held accounts are kept and shown, but weigh nothing in the
// feed's popularity formula.
func TestHeldAccountsReactionsCarryNoWeightInTheFeed(t *testing.T) {
	f := newHeldFixture(t, defaultPolicy())
	wall := f.community(t, f.invited)
	post, err := f.posts.Create(f.ctx, posts.CreateInput{CommunityID: wall, Text: "rank me"}, posts.ActorMeta{UserID: f.invited, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}

	react := func(user uuid.UUID) {
		t.Helper()
		if _, err := f.pool.Exec(f.ctx,
			`INSERT INTO reactions (post_id, user_id, emoji) VALUES ($1, $2, '👍')`, post.ID, user); err != nil {
			t.Fatal(err)
		}
	}
	repo := posts.NewPostgresRepository()
	reactorsOf := func(hold time.Duration) int {
		t.Helper()
		inputs, err := repo.ScoreInputs(f.ctx, f.pool, time.Now().Add(-time.Hour), hold)
		if err != nil {
			t.Fatal(err)
		}
		for _, in := range inputs {
			if in.PostID == post.ID {
				return in.Reactors
			}
		}
		t.Fatal("the post is not in the ranking inputs")
		return 0
	}

	react(f.fresh)
	if n := reactorsOf(24 * time.Hour); n != 0 {
		t.Fatalf("a held account's reaction counted: reactors = %d, want 0", n)
	}
	// The reaction itself is stored — it is the weight that is zero.
	var stored int
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM reactions WHERE post_id = $1`, post.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 1 {
		t.Fatalf("reactions stored = %d, want 1", stored)
	}

	react(f.mature)
	react(f.invited)
	if n := reactorsOf(24 * time.Hour); n != 2 {
		t.Fatalf("reactors = %d, want 2 (the mature and the invited one)", n)
	}
	// With the quarantine switched off every reaction counts again.
	if n := reactorsOf(0); n != 3 {
		t.Fatalf("quarantine off: reactors = %d, want 3", n)
	}
}

// What /users/me tells the screens, so they can say "opens in N hours"
// instead of letting someone fill a form that will be refused.
func TestStatusTellsTheScreenWhatIsHeld(t *testing.T) {
	f := newHeldFixture(t, defaultPolicy())
	status, err := f.checker.StatusFor(f.ctx, f.fresh)
	if err != nil || status == nil {
		t.Fatalf("status of a fresh account: %+v, %v", status, err)
	}
	if status.HoursLeft != 23 || status.NewChatsPerDay != 3 || status.MaxUploadBytes != 2<<20 {
		t.Fatalf("status = %+v", status)
	}
	for name, id := range map[string]uuid.UUID{"mature": f.mature, "invited": f.invited} {
		if s, err := f.checker.StatusFor(f.ctx, id); err != nil || s != nil {
			t.Errorf("%s: status %+v, %v — want none", name, s, err)
		}
	}
}

// A deployment that switched the hold off keeps everything open.
func TestQuarantineOffHoldsNothing(t *testing.T) {
	f := newHeldFixture(t, config.QuarantineConfig{})
	wall := f.community(t, f.invited)
	f.join(t, wall, f.fresh)
	if _, err := f.posts.Create(f.ctx, posts.CreateInput{CommunityID: wall, Text: "hello"}, posts.ActorMeta{UserID: f.fresh}); err != nil {
		t.Fatalf("publishing with the hold off: %v", err)
	}
	if _, err := f.files.Upload(f.ctx, "big.bin", make([]byte, (2<<20)+1), f.fresh, 0, attachments.Meta{}); err != nil {
		t.Fatalf("big upload with the hold off: %v", err)
	}
}
