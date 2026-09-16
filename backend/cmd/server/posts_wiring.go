package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/blobstore"
	"kisy-backend/internal/posts"
)

// The adapters that let internal/posts use the rest of the app without
// importing it. Each one lives here, in the composition root, so the rules
// they borrow stay written once in the module that owns them.

// postsCommunities answers "what is this community, to this person?" using the
// groups service — which is where group visibility, membership and the
// post policy are already decided.
type postsCommunities struct {
	groups *groups.Service
}

func (c postsCommunities) actor(a posts.ActorMeta) groups.ActorMeta {
	return groups.ActorMeta{UserID: a.UserID, RoleLevel: a.RoleLevel, SessionID: a.SessionID}
}

func (c postsCommunities) Resolve(
	ctx context.Context, communityID uuid.UUID, actor posts.ActorMeta,
) (posts.CommunityView, error) {
	g, err := c.groups.Get(ctx, communityID, c.actor(actor))
	if err != nil {
		return posts.CommunityView{}, mapGroupErr(err)
	}
	// Viewer adds membership and the post right on top of the same visibility
	// check, so a non-member reading a public community's wall still gets a
	// truthful "you are not in this one".
	vs, err := c.groups.Viewer(ctx, communityID, c.actor(actor))
	if err != nil {
		return posts.CommunityView{}, mapGroupErr(err)
	}
	return posts.CommunityView{
		ID:         g.ID,
		Name:       g.Name,
		AvatarURL:  g.AvatarURL,
		Kind:       g.Kind,
		IsPublic:   g.IsPublic,
		JoinPolicy: g.JoinPolicy,
		IsMember:   vs.Member,
		Verified:   g.VerifiedAt != nil,
		CanPost:    vs.CanPost,
	}, nil
}

func (c postsCommunities) ResolveMany(
	ctx context.Context, ids []uuid.UUID, actor posts.ActorMeta,
) (map[uuid.UUID]posts.CommunityView, error) {
	out := make(map[uuid.UUID]posts.CommunityView, len(ids))
	for _, id := range ids {
		if _, done := out[id]; done {
			continue // a page is usually a handful of communities, repeated
		}
		view, err := c.Resolve(ctx, id, actor)
		if errors.Is(err, posts.ErrNotFound) {
			continue // filtered out of the page by the caller
		}
		if err != nil {
			return nil, err
		}
		out[id] = view
	}
	return out, nil
}

func (c postsCommunities) MemberIDs(ctx context.Context, communityID uuid.UUID) ([]uuid.UUID, error) {
	return c.groups.MemberIDs(ctx, communityID)
}

// A community someone may not see must be indistinguishable from one that does
// not exist — otherwise its existence leaks.
func mapGroupErr(err error) error {
	if errors.Is(err, groups.ErrNotFound) || errors.Is(err, groups.ErrNotMember) {
		return posts.ErrNotFound
	}
	return err
}

// postsPublisher announces a new post over the WebSocket.
//
// Its own port, not the message publisher: a post is not a message, and this
// is the seam that keeps a reaction on a post from ever surfacing in a chat.
type postsPublisher struct {
	publish func(memberIDs []uuid.UUID, communityID, postID uuid.UUID)
}

func (p postsPublisher) PublishPost(memberIDs []uuid.UUID, communityID, postID uuid.UUID) {
	if p.publish != nil {
		p.publish(memberIDs, communityID, postID)
	}
}

// postsMedia adapts the blob store to what posts needs: a name and a content
// type in, a storage key out.
type postsMedia struct {
	store blobstore.Store
}

func (m postsMedia) Put(ctx context.Context, name, mime string, raw []byte) (string, error) {
	key := "posts/" + uuid.NewString()
	if err := m.store.Put(ctx, key, raw, mime); err != nil {
		return "", err
	}
	return key, nil
}

func (m postsMedia) Get(ctx context.Context, storagePath string) ([]byte, string, error) {
	raw, err := m.store.Get(ctx, storagePath)
	if err != nil {
		return nil, "", err
	}
	// The row carries the content type; the store does not return one.
	return raw, "", nil
}
