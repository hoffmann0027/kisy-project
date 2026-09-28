//go:build integration

package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/posts"
)

// Audit A-01: a closed community's wall was readable by anyone who could see
// the community in the directory — including a freshly self-registered basic
// account. These tests go through the real composition-root adapter
// (postsCommunities), because that is where visibility and membership meet;
// a fake would test the fake.

type wallFixture struct {
	ctx     context.Context
	pool    *pgxpool.Pool
	gsvc    *groups.Service
	psvc    *posts.Service
	repo    posts.Repository
	founder posts.ActorMeta
}

func newWallFixture(t *testing.T) (wallFixture, func(name string, level int) posts.ActorMeta) {
	t.Helper()
	pool := testdb.New(t)
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	gsvc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	repo := posts.NewPostgresRepository()
	psvc := posts.NewService(pool, repo, postsCommunities{groups: gsvc}, rec)

	seed := func(name string, level int) posts.ActorMeta {
		return posts.ActorMeta{UserID: testdb.SeedUser(t, pool, name, level), RoleLevel: level}
	}
	return wallFixture{ctx: context.Background(), pool: pool, gsvc: gsvc, psvc: psvc, repo: repo, founder: seed("founder", 5)}, seed
}

// community creates a community with no clearance threshold (visible to every
// account, basic ones included — the exposure that made A-01 external) and
// one post carrying one inline media file.
func (f wallFixture) community(t *testing.T, public bool) (groupID, postID, mediaID uuid.UUID) {
	t.Helper()
	g, err := f.gsvc.Create(f.ctx, groups.CreateInput{Name: "Wall", Kind: groups.KindCommunity, IsPublic: public},
		groups.ActorMeta{UserID: f.founder.UserID, RoleLevel: f.founder.RoleLevel})
	if err != nil {
		t.Fatal(err)
	}
	p := &posts.Post{CommunityID: g.ID, AuthorID: f.founder.UserID, Text: "members only"}
	if err := f.repo.Create(f.ctx, f.pool, p); err != nil {
		t.Fatal(err)
	}
	dto, err := f.psvc.AttachMedia(f.ctx, p.ID, posts.UploadedFile{FileName: "a.txt", Bytes: []byte("secret file")}, f.founder)
	if err != nil {
		t.Fatal(err)
	}
	return g.ID, p.ID, dto.Media[0].ID
}

func TestClosedCommunityWallIsForMembersOnly(t *testing.T) {
	f, seed := newWallFixture(t)
	outsider := seed("outsider", 0) // self-registered, no invitation
	staff := seed("staff", 8)       // invited, still not a member
	groupID, postID, mediaID := f.community(t, false)

	// Precondition that makes this reachable: the outsider can SEE the community.
	if _, err := f.gsvc.Get(f.ctx, groupID, groups.ActorMeta{UserID: outsider.UserID}); err != nil {
		t.Fatalf("precondition: a threshold-free community is visible to a basic account, got %v", err)
	}

	for name, actor := range map[string]posts.ActorMeta{"basic non-member": outsider, "invited non-member": staff} {
		if _, err := f.psvc.ListCommunity(f.ctx, groupID, "", 20, actor); !errors.Is(err, posts.ErrMembersOnly) {
			t.Errorf("%s: wall of a closed community: want ErrMembersOnly, got %v", name, err)
		}
		if _, _, _, err := f.psvc.ReadMedia(f.ctx, mediaID, actor); !errors.Is(err, posts.ErrMembersOnly) {
			t.Errorf("%s: media of a closed community: want ErrMembersOnly, got %v", name, err)
		}
		if err := f.psvc.React(f.ctx, postID, "👍", true, actor); !errors.Is(err, posts.ErrMembersOnly) {
			t.Errorf("%s: reacting in a closed community: want ErrMembersOnly, got %v", name, err)
		}
	}
}

func TestClosedCommunityWallStaysOpenToMembersAndTheCEO(t *testing.T) {
	f, seed := newWallFixture(t)
	ceo := seed("chief", 1)
	groupID, postID, mediaID := f.community(t, false)

	for name, actor := range map[string]posts.ActorMeta{"founder (member)": f.founder, "CEO (moderation)": ceo} {
		page, err := f.psvc.ListCommunity(f.ctx, groupID, "", 20, actor)
		if err != nil || len(page.Posts) != 1 {
			t.Errorf("%s: wall: want 1 post, got %d posts, err %v", name, len(page.Posts), err)
		}
		if _, _, _, err := f.psvc.ReadMedia(f.ctx, mediaID, actor); err != nil {
			t.Errorf("%s: media: %v", name, err)
		}
		if err := f.psvc.React(f.ctx, postID, "🔥", true, actor); err != nil {
			t.Errorf("%s: react: %v", name, err)
		}
	}
}

func TestPublicCommunityWallIsReadableByNonMembers(t *testing.T) {
	f, seed := newWallFixture(t)
	outsider := seed("outsider", 0)
	groupID, postID, mediaID := f.community(t, true)

	if page, err := f.psvc.ListCommunity(f.ctx, groupID, "", 20, outsider); err != nil || len(page.Posts) != 1 {
		t.Fatalf("public wall: want 1 post, got %v / %v", page.Posts, err)
	}
	if _, _, _, err := f.psvc.ReadMedia(f.ctx, mediaID, outsider); err != nil {
		t.Fatalf("public media: %v", err)
	}
	if err := f.psvc.React(f.ctx, postID, "👍", true, outsider); err != nil {
		t.Fatalf("public react: %v", err)
	}
}

// Audit A-13: in a community where every member may publish, the right to
// write carried the right to erase — any member could delete anyone's post,
// and there is no undo in the app. Deleting someone else's post belongs to
// whoever runs the community.
func TestOnlyTheAuthorAndTheCommunitysOwnPeopleDeleteAPost(t *testing.T) {
	f, seed := newWallFixture(t)
	member := seed("plain_member", 7)
	groupID, postID, _ := f.community(t, true)

	// Anyone may publish here, which is exactly the case A-13 was about.
	if _, err := f.gsvc.SetPolicies(f.ctx, groupID, groups.PolicyJoinOpen, groups.PolicyPostAll,
		groups.ActorMeta{UserID: f.founder.UserID, RoleLevel: f.founder.RoleLevel}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.gsvc.Join(f.ctx, groupID, groups.ActorMeta{UserID: member.UserID, RoleLevel: member.RoleLevel}); err != nil {
		t.Fatal(err)
	}

	// The member may write...
	own, err := f.psvc.Create(f.ctx, posts.CreateInput{CommunityID: groupID, Text: "мой пост"}, member)
	if err != nil {
		t.Fatalf("a member of an open community must be able to publish: %v", err)
	}
	// ...but not erase the founder's post.
	if err := f.psvc.Delete(f.ctx, postID, member); !errors.Is(err, posts.ErrForbidden) {
		t.Fatalf("a plain member deleted someone else's post: %v", err)
	}
	// Their own, yes.
	if err := f.psvc.Delete(f.ctx, own.ID, member); err != nil {
		t.Fatalf("deleting their own post: %v", err)
	}
	// And the founder still moderates the wall.
	if err := f.psvc.Delete(f.ctx, postID, f.founder); err != nil {
		t.Fatalf("the founder deleting a post on their own wall: %v", err)
	}
}
