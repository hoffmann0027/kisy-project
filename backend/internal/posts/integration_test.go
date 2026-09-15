//go:build integration

package posts_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/posts"
)

// A reaction to a post is a vote, and a vote is cast once (migration 45).
func TestOneReactionPerPersonPerPost(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	gsvc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	repo := posts.NewPostgresRepository()

	editor := testdb.SeedUser(t, pool, "editor", 5)
	reader := testdb.SeedUser(t, pool, "reader", 8)
	other := testdb.SeedUser(t, pool, "other", 8)

	c, err := gsvc.Create(ctx, groups.CreateInput{Name: "Wall", Kind: groups.KindCommunity, IsPublic: true},
		groups.ActorMeta{UserID: editor, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	p := &posts.Post{CommunityID: c.ID, AuthorID: editor, Text: "hello"}
	if err := repo.Create(ctx, pool, p); err != nil {
		t.Fatal(err)
	}

	mustReact := func(user uuid.UUID, emoji string) {
		t.Helper()
		if err := repo.AddReaction(ctx, pool, p.ID, user, emoji); err != nil {
			t.Fatalf("react %s: %v", emoji, err)
		}
	}
	summary := func() map[string]posts.ReactionSummary {
		t.Helper()
		got, err := repo.ReactionsFor(ctx, pool, []uuid.UUID{p.ID}, reader)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]posts.ReactionSummary{}
		for _, s := range got[p.ID] {
			out[s.Emoji] = s
		}
		return out
	}

	mustReact(reader, "👍")
	mustReact(reader, "🔥")
	mustReact(reader, "🔥") // repeating the same choice changes nothing
	got := summary()
	if len(got) != 1 || got["🔥"].Count != 1 || !got["🔥"].Mine {
		t.Fatalf("a second emoji must replace the first, got %+v", got)
	}

	// Someone else's reaction is theirs: it neither replaces nor is replaced.
	mustReact(other, "👍")
	got = summary()
	if got["🔥"].Count != 1 || got["👍"].Count != 1 || got["👍"].Mine {
		t.Fatalf("two people, two reactions, got %+v", got)
	}

	// Taking back an emoji you have already swapped away is a no-op: a late
	// "remove 👍" from a slow client must not delete the 🔥 that replaced it.
	if err := repo.RemoveReaction(ctx, pool, p.ID, reader, "👍"); err != nil {
		t.Fatal(err)
	}
	if got = summary(); !got["🔥"].Mine {
		t.Fatalf("a stale removal deleted the current reaction, got %+v", got)
	}
	if err := repo.RemoveReaction(ctx, pool, p.ID, reader, "🔥"); err != nil {
		t.Fatal(err)
	}
	got = summary()
	if _, still := got["🔥"]; still || got["👍"].Count != 1 {
		t.Fatalf("removal must take exactly the reader's reaction, got %+v", got)
	}
}
