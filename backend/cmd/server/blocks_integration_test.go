//go:build integration

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/blocks"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/posts"
	"kisy-backend/internal/users"
)

// Blocking, through the real services wired the way the server wires them
// (wireBlocks). What a block must do: end the conversation both ways, keep the
// two out of each other's feed and search, and never say who blocked whom.

type blockFixture struct {
	ctx      context.Context
	pool     *pgxpool.Pool
	blocks   *blocks.Service
	chats    *chats.Service
	msgs     *messages.Service
	posts    *posts.Service
	groups   *groups.Service
	users    *users.Service
	me, them uuid.UUID
}

func newBlockFixture(t *testing.T) blockFixture {
	t.Helper()
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(quiet())
	suffix := uuid.NewString()[:8]

	f := blockFixture{ctx: ctx, pool: pool}
	f.me = testdb.SeedUser(t, pool, "me_"+suffix, 0)
	f.them = testdb.SeedUser(t, pool, "them_"+suffix, 0)

	f.blocks = blocks.NewService(pool, rec)
	f.chats = chats.NewService(pool, chats.NewPostgresRepository(),
		func(context.Context, uuid.UUID) (int, bool) { return 0, true })
	f.msgs = messages.NewService(pool, messages.NewPostgresRepository(), rec, messages.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			if ok, err := f.chats.IsParticipant(ctx, chatID, actorID); err != nil || !ok {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(context.Context, uuid.UUID, uuid.UUID, int) error { return messages.ErrNotFound },
	})
	f.groups = groups.NewService(pool, groups.NewPostgresRepository(), rec)
	f.posts = posts.NewService(pool, posts.NewPostgresRepository(), postsCommunities{groups: f.groups}, rec)
	f.users = users.NewService(pool, users.NewPostgresRepository(), rec)

	// Calls are wired in the server too; here the three that need a database.
	f.chats.SetBlockCheck(f.blocks.Between)
	f.msgs.SetBlockCheck(func(ctx context.Context, chatID, senderID uuid.UUID) (bool, error) {
		ids, err := f.chats.ParticipantIDs(ctx, chatID)
		if err != nil || len(ids) != 2 {
			return false, err
		}
		other := ids[0]
		if other == senderID {
			other = ids[1]
		}
		return f.blocks.Between(ctx, senderID, other)
	})
	return f
}

func (f blockFixture) block(t *testing.T, blocker, blocked uuid.UUID) {
	t.Helper()
	if err := f.blocks.Add(f.ctx, blocks.ActorMeta{UserID: blocker}, blocked); err != nil {
		t.Fatal(err)
	}
}

func (f blockFixture) send(sender uuid.UUID, chatID uuid.UUID) error {
	alg, epoch := int16(1), int64(1)
	_, err := f.msgs.Send(f.ctx, messages.SendInput{
		ChatType: "private", ChatID: chatID, Ciphertext: []byte("ciphertext"), Alg: &alg, Epoch: &epoch,
	}, messages.ActorMeta{UserID: sender})
	return err
}

func TestABlockEndsTheConversationBothWays(t *testing.T) {
	f := newBlockFixture(t)
	chat, err := f.chats.OpenPrivateChat(f.ctx, f.them, chats.ActorMeta{UserID: f.me})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.send(f.me, chat.ID); err != nil {
		t.Fatalf("before the block: %v", err)
	}

	f.block(t, f.me, f.them)

	// The blocked person cannot write...
	if err := f.send(f.them, chat.ID); !errors.Is(err, messages.ErrBlocked) {
		t.Fatalf("the blocked person wrote anyway: %v", err)
	}
	// ...and neither does the blocker keep the one-sided conversation going.
	if err := f.send(f.me, chat.ID); !errors.Is(err, messages.ErrBlocked) {
		t.Fatalf("the blocker wrote into a blocked chat: %v", err)
	}

	// Lifting it restores both directions.
	if err := f.blocks.Remove(f.ctx, blocks.ActorMeta{UserID: f.me}, f.them); err != nil {
		t.Fatal(err)
	}
	if err := f.send(f.them, chat.ID); err != nil {
		t.Fatalf("after unblocking: %v", err)
	}
}

func TestABlockedPersonCannotStartAConversation(t *testing.T) {
	f := newBlockFixture(t)
	f.block(t, f.me, f.them)

	if _, err := f.chats.OpenPrivateChat(f.ctx, f.me, chats.ActorMeta{UserID: f.them}); !errors.Is(err, chats.ErrBlocked) {
		t.Fatalf("the blocked person opened a chat: %v", err)
	}
	// And the blocker cannot start one either — a block is not a mute.
	if _, err := f.chats.OpenPrivateChat(f.ctx, f.them, chats.ActorMeta{UserID: f.me}); !errors.Is(err, chats.ErrBlocked) {
		t.Fatalf("the blocker opened a chat: %v", err)
	}
	// Someone else is unaffected.
	third := testdb.SeedUser(t, f.pool, "third_"+uuid.NewString()[:8], 0)
	if _, err := f.chats.OpenPrivateChat(f.ctx, third, chats.ActorMeta{UserID: f.them}); err != nil {
		t.Fatalf("an unrelated conversation: %v", err)
	}
}

func TestBlockedAccountsDisappearFromTheFeedAndSearch(t *testing.T) {
	f := newBlockFixture(t)
	// A community anyone may post in, and a post by the person to be blocked.
	actor := groups.ActorMeta{UserID: f.them, RoleLevel: 5}
	g, err := f.groups.Create(f.ctx, groups.CreateInput{Name: "Wall " + uuid.NewString()[:6], Kind: groups.KindCommunity, IsPublic: true}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.groups.SetPolicies(f.ctx, g.ID, groups.PolicyJoinOpen, groups.PolicyPostAll, actor); err != nil {
		t.Fatal(err)
	}
	post, err := f.posts.Create(f.ctx, posts.CreateInput{CommunityID: g.ID, Text: "их пост"}, posts.ActorMeta{UserID: f.them, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}

	inFeed := func() bool {
		t.Helper()
		page, err := f.posts.Feed(f.ctx, "new", "", 50, posts.ActorMeta{UserID: f.me})
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range page.Posts {
			if p.ID == post.ID {
				return true
			}
		}
		return false
	}
	found := func(viewer uuid.UUID, viewerLevel int, needle string) bool {
		t.Helper()
		list, err := f.users.Directory(f.ctx, viewer, viewerLevel, needle, 50)
		if err != nil {
			t.Fatal(err)
		}
		return len(list) > 0
	}
	theirName := func() string {
		t.Helper()
		var name string
		if err := f.pool.QueryRow(f.ctx, `SELECT display_name FROM users WHERE id = $1`, f.them).Scan(&name); err != nil {
			t.Fatal(err)
		}
		return name
	}()
	myName := func() string {
		t.Helper()
		var name string
		if err := f.pool.QueryRow(f.ctx, `SELECT display_name FROM users WHERE id = $1`, f.me).Scan(&name); err != nil {
			t.Fatal(err)
		}
		return name
	}()

	if !inFeed() {
		t.Fatal("precondition: the post is in the feed before the block")
	}
	if !found(f.me, 0, theirName) {
		t.Fatal("precondition: the account is findable before the block")
	}

	f.block(t, f.me, f.them)

	if inFeed() {
		t.Error("a blocked account's post is still in the feed")
	}
	if found(f.me, 0, theirName) {
		t.Error("the blocker still finds the account they blocked")
	}
	// And the block hides the blocker from the blocked person as well.
	if found(f.them, 0, myName) {
		t.Error("the blocked person still finds the one who blocked them")
	}
}

// The audit journal records both the block and its lifting.
func TestBlockingIsAudited(t *testing.T) {
	f := newBlockFixture(t)
	f.block(t, f.me, f.them)
	if err := f.blocks.Remove(f.ctx, blocks.ActorMeta{UserID: f.me}, f.them); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.pool.QueryRow(f.ctx,
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1 AND action IN ('user.block', 'user.unblock')`,
		f.me).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("audit entries = %d, want 2", n)
	}
}

func TestBlockingYourselfIsRefusedAndBlockingTwiceIsFine(t *testing.T) {
	f := newBlockFixture(t)
	if err := f.blocks.Add(f.ctx, blocks.ActorMeta{UserID: f.me}, f.me); !errors.Is(err, blocks.ErrSelf) {
		t.Fatalf("blocking yourself: %v", err)
	}
	f.block(t, f.me, f.them)
	f.block(t, f.me, f.them)
	list, err := f.blocks.List(f.ctx, f.me)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].UserID != f.them || list[0].CreatedAt.After(time.Now()) {
		t.Fatalf("list = %+v", list)
	}
}

// A block also ends what the two show each other under other people's posts:
// seeing a heart from the person you blocked is what blocking was meant to end.
func TestBlockedAccountsReactionsAreNotShown(t *testing.T) {
	f := newBlockFixture(t)
	actor := groups.ActorMeta{UserID: f.me, RoleLevel: 5}
	g, err := f.groups.Create(f.ctx, groups.CreateInput{Name: "Wall " + uuid.NewString()[:6], Kind: groups.KindCommunity, IsPublic: true}, actor)
	if err != nil {
		t.Fatal(err)
	}
	post, err := f.posts.Create(f.ctx, posts.CreateInput{CommunityID: g.ID, Text: "пост"}, posts.ActorMeta{UserID: f.me, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx,
		`INSERT INTO reactions (post_id, user_id, emoji) VALUES ($1, $2, '👍')`, post.ID, f.them); err != nil {
		t.Fatal(err)
	}

	reactionsOn := func(viewer uuid.UUID) int {
		t.Helper()
		page, err := f.posts.ListCommunity(f.ctx, g.ID, "", 10, posts.ActorMeta{UserID: viewer, RoleLevel: 5})
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range page.Posts {
			if p.ID == post.ID {
				total := 0
				for _, r := range p.Reactions {
					total += r.Count
				}
				return total
			}
		}
		t.Fatal("the post is not on its own wall")
		return 0
	}

	if reactionsOn(f.me) != 1 {
		t.Fatal("precondition: the reaction is visible before the block")
	}
	f.block(t, f.me, f.them)
	if n := reactionsOn(f.me); n != 0 {
		t.Fatalf("a blocked account's reaction is still shown: %d", n)
	}
	// Everyone else keeps seeing it — a block is between two people.
	third := testdb.SeedUser(t, f.pool, "third_"+uuid.NewString()[:8], 5)
	if n := reactionsOn(third); n != 1 {
		t.Fatalf("the reaction disappeared for an unrelated viewer: %d", n)
	}
}
