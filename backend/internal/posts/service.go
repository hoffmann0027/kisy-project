package posts

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
)

// CommunityView is a community as one particular actor sees it.
type CommunityView struct {
	ID         uuid.UUID
	Name       string
	AvatarURL  *string
	Kind       string
	IsPublic   bool
	JoinPolicy string
	IsMember   bool
	// CanPost follows the group's own post policy and in-group role.
	CanPost bool
}

// Communities is the slice of the groups module that posts needs, injected in
// the composition root so this package does not import groups (and so the
// access rules stay written once, where they already are).
type Communities interface {
	// Resolve returns the community as this actor sees it, or ErrNotFound when
	// they may not see it at all — a hidden community and a missing one must
	// be indistinguishable.
	Resolve(ctx context.Context, communityID uuid.UUID, actor ActorMeta) (CommunityView, error)
	// ResolveMany does the same for a page of posts without a query per row.
	ResolveMany(ctx context.Context, ids []uuid.UUID, actor ActorMeta) (map[uuid.UUID]CommunityView, error)
	// MemberIDs lists who should be told about a new post.
	MemberIDs(ctx context.Context, communityID uuid.UUID) ([]uuid.UUID, error)
}

// Profiles renders author cards.
type Profiles interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]AuthorCard, error)
}

// MediaStore persists an uploaded file and reads it back. Satisfied by the
// blob store; a local interface keeps posts out of that dependency.
type MediaStore interface {
	Put(ctx context.Context, name, mime string, raw []byte) (storagePath string, err error)
	Get(ctx context.Context, storagePath string) (raw []byte, mime string, err error)
}

// UploadedFile is one file on its way onto a post.
type UploadedFile struct {
	FileName string
	MimeType string
	Bytes    []byte
}

// Publisher announces a new post to a community's members.
//
// Deliberately its own port rather than the chat publisher: a post is not a
// message, and a reaction to a post must never arrive in someone's chat as a
// message reaction. Keeping the two apart here is what makes that impossible
// rather than merely unlikely.
type Publisher interface {
	PublishPost(memberIDs []uuid.UUID, communityID, postID uuid.UUID)
}

type Service struct {
	pool        *pgxpool.Pool
	repo        Repository
	communities Communities
	profiles    Profiles
	audit       audit.Recorder
	media       MediaStore
	pub         Publisher
	ranker      Ranker
}

func NewService(pool *pgxpool.Pool, repo Repository, communities Communities, rec audit.Recorder) *Service {
	return &Service{pool: pool, repo: repo, communities: communities, audit: rec}
}

func (s *Service) SetProfiles(p Profiles)     { s.profiles = p }
func (s *Service) SetPublisher(p Publisher)   { s.pub = p }
func (s *Service) SetRanker(r Ranker)         { s.ranker = r }
func (s *Service) SetMediaStore(m MediaStore) { s.media = m }

// CreateInput is validated by the handler before it reaches the service.
type CreateInput struct {
	CommunityID uuid.UUID
	Text        string
	Media       []Media
}

// Create publishes a post to a community's wall.
func (s *Service) Create(ctx context.Context, in CreateInput, actor ActorMeta) (*DTO, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" && len(in.Media) == 0 {
		return nil, ErrEmpty
	}
	if len(text) > MaxTextLength {
		return nil, ErrTooLong
	}
	if len(in.Media) > MaxMediaPerPost {
		return nil, ErrTooLong
	}
	for _, m := range in.Media {
		if !validMediaKind(m.Kind) {
			return nil, ErrEmpty
		}
	}

	community, err := s.communities.Resolve(ctx, in.CommunityID, actor)
	if err != nil {
		return nil, err
	}
	if community.Kind != "community" {
		return nil, ErrNotCommunity
	}
	// Reading a community and writing to it are different rights: members read
	// the wall, its editors write to it.
	if !community.CanPost {
		return nil, ErrForbidden
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("posts: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	p := &Post{CommunityID: in.CommunityID, AuthorID: actor.UserID, Text: text}
	if err := s.repo.Create(ctx, tx, p); err != nil {
		return nil, err
	}
	if len(in.Media) > 0 {
		if err := s.repo.AddMedia(ctx, tx, p.ID, in.Media); err != nil {
			return nil, err
		}
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     "post.created",
		TargetType: "post",
		TargetID:   &p.ID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"communityId": in.CommunityID},
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("posts: commit: %w", err)
	}

	if s.pub != nil {
		if members, err := s.communities.MemberIDs(ctx, in.CommunityID); err == nil {
			s.pub.PublishPost(members, in.CommunityID, p.ID)
		}
	}

	page, err := s.render(ctx, []Post{*p}, actor)
	if err != nil || len(page) == 0 {
		return nil, err
	}
	return &page[0], nil
}

// Delete removes a post. Its author may always delete it; so may anyone who
// could have posted it in the first place — the community's editors.
func (s *Service) Delete(ctx context.Context, postID uuid.UUID, actor ActorMeta) error {
	p, err := s.repo.Get(ctx, s.pool, postID)
	if err != nil {
		return err
	}
	community, err := s.communities.Resolve(ctx, p.CommunityID, actor)
	if err != nil {
		return err
	}
	if p.AuthorID != actor.UserID && !community.CanPost {
		return ErrForbidden
	}
	if err := s.repo.SoftDelete(ctx, s.pool, postID, time.Now().UTC()); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, s.pool, audit.Event{
		ActorID:    &actor.UserID,
		Action:     "post.deleted",
		TargetType: "post",
		TargetID:   &postID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
	})
	return nil
}

// ListCommunity returns one community's wall.
func (s *Service) ListCommunity(
	ctx context.Context, communityID uuid.UUID, cursor string, limit int, actor ActorMeta,
) (Page, error) {
	if _, err := s.communities.Resolve(ctx, communityID, actor); err != nil {
		return Page{}, err
	}
	before, err := decodeCursor(cursor)
	if err != nil {
		return Page{}, err
	}
	rows, err := s.repo.ListByCommunity(ctx, s.pool, communityID, before, limit+1)
	if err != nil {
		return Page{}, err
	}
	return s.page(ctx, rows, limit, actor)
}

// Feed returns the shared feed across public communities. sort is "popular"
// (the ranking) or "new" (reverse chronological).
func (s *Service) Feed(ctx context.Context, sort, cursor string, limit int, actor ActorMeta) (Page, error) {
	if sort == "new" || s.ranker == nil {
		before, err := decodeCursor(cursor)
		if err != nil {
			return Page{}, err
		}
		rows, err := s.repo.ListNewest(ctx, s.pool, actor.UserID, actor.RoleLevel, before, limit+1)
		if err != nil {
			return Page{}, err
		}
		return s.page(ctx, rows, limit, actor)
	}

	// Popular: the ranking is global, and what each reader may see is not, so
	// the page is filtered after it is ranked. A reader who hid many
	// communities can therefore get a short page — the cursor still advances
	// by what was asked for, so paging never stalls.
	offset := decodeOffset(cursor)
	ids, err := s.ranker.Top(ctx, offset, limit)
	if err != nil {
		return Page{}, err
	}
	rows, err := s.repo.ByIDsVisible(ctx, s.pool, ids, actor.UserID, actor.RoleLevel)
	if err != nil {
		return Page{}, err
	}
	dtos, err := s.render(ctx, rows, actor)
	if err != nil {
		return Page{}, err
	}
	next := ""
	if len(ids) == limit {
		next = encodeOffset(offset + limit)
	}
	return Page{Posts: dtos, NextCursor: next}, nil
}

// React adds or removes the actor's reaction to a post.
func (s *Service) React(ctx context.Context, postID uuid.UUID, emoji string, on bool, actor ActorMeta) error {
	if emoji == "" || len(emoji) > 32 {
		return ErrEmpty
	}
	p, err := s.repo.Get(ctx, s.pool, postID)
	if err != nil {
		return err
	}
	// Seeing the community is enough to react to its posts; posting is not.
	if _, err := s.communities.Resolve(ctx, p.CommunityID, actor); err != nil {
		return err
	}
	if on {
		return s.repo.AddReaction(ctx, s.pool, postID, actor.UserID, emoji)
	}
	return s.repo.RemoveReaction(ctx, s.pool, postID, actor.UserID, emoji)
}

// AttachMedia adds one uploaded file to a post and returns the updated post.
//
// Only the author may attach, and only to a post that is still theirs: media
// travels after the post because a file and a JSON body do not share a
// request, not because anyone else should be able to add to it later.
func (s *Service) AttachMedia(
	ctx context.Context, postID uuid.UUID, file UploadedFile, actor ActorMeta,
) (*DTO, error) {
	if len(file.Bytes) == 0 {
		return nil, ErrEmpty
	}
	p, err := s.repo.Get(ctx, s.pool, postID)
	if err != nil {
		return nil, err
	}
	if p.AuthorID != actor.UserID {
		return nil, ErrForbidden
	}
	if _, err := s.communities.Resolve(ctx, p.CommunityID, actor); err != nil {
		return nil, err
	}

	existing, err := s.repo.MediaFor(ctx, s.pool, []uuid.UUID{postID})
	if err != nil {
		return nil, err
	}
	if len(existing[postID]) >= MaxMediaPerPost {
		return nil, ErrTooLong
	}

	name := strings.TrimSpace(file.FileName)
	if name == "" {
		name = "file"
	}
	m := Media{
		Kind:      mediaKindFor(file.MimeType),
		FileName:  name,
		MimeType:  file.MimeType,
		SizeBytes: int64(len(file.Bytes)),
		Position:  len(existing[postID]),
	}
	// With an object store the bytes go there; without one they stay in the
	// row. The alternative — refusing attachments on a deployment that has no
	// bucket — would make the feature depend on infrastructure the user never
	// asked about.
	if s.media != nil {
		path, err := s.media.Put(ctx, name, file.MimeType, file.Bytes)
		if err != nil {
			return nil, fmt.Errorf("posts: store media: %w", err)
		}
		m.StoragePath = path
	} else {
		m.Bytes = file.Bytes
	}
	if err := s.repo.AddMedia(ctx, s.pool, postID, []Media{m}); err != nil {
		return nil, err
	}

	page, err := s.render(ctx, []Post{*p}, actor)
	if err != nil || len(page) == 0 {
		return nil, err
	}
	return &page[0], nil
}

// ReadMedia streams one post attachment back, after checking that the reader
// may see the post it hangs off.
func (s *Service) ReadMedia(ctx context.Context, mediaID uuid.UUID, actor ActorMeta) ([]byte, string, error) {
	m, postID, err := s.repo.MediaByID(ctx, s.pool, mediaID)
	if err != nil {
		return nil, "", err
	}
	p, err := s.repo.Get(ctx, s.pool, postID)
	if err != nil {
		return nil, "", err
	}
	// The community decides: a file is exactly as visible as its post.
	if _, err := s.communities.Resolve(ctx, p.CommunityID, actor); err != nil {
		return nil, "", err
	}
	// Whichever half of the XOR this row uses (migration 44).
	if len(m.Bytes) > 0 {
		return m.Bytes, m.MimeType, nil
	}
	if s.media == nil {
		// The row points at an object store this process does not have.
		return nil, "", ErrNotFound
	}
	raw, mime, err := s.media.Get(ctx, m.StoragePath)
	if err != nil {
		return nil, "", fmt.Errorf("posts: read media: %w", err)
	}
	if mime == "" {
		mime = m.MimeType
	}
	return raw, mime, nil
}

// mediaKindFor maps a content type to the kind stored on the row, which is
// what the client uses to pick a renderer.
func mediaKindFor(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return MediaImage
	case strings.HasPrefix(mime, "video/"):
		return MediaVideo
	case strings.HasPrefix(mime, "audio/"):
		return MediaAudio
	default:
		return MediaFile
	}
}

// HideCommunity drops a community out of this user's feed (and ShowCommunity
// puts it back). Membership is untouched — this is about the feed only.
func (s *Service) HideCommunity(ctx context.Context, communityID uuid.UUID, hidden bool, actor ActorMeta) error {
	if hidden {
		return s.repo.HideCommunity(ctx, s.pool, actor.UserID, communityID)
	}
	return s.repo.ShowCommunity(ctx, s.pool, actor.UserID, communityID)
}

// page turns an over-fetched row set into a page plus its cursor.
func (s *Service) page(ctx context.Context, rows []Post, limit int, actor ActorMeta) (Page, error) {
	next := ""
	if len(rows) > limit {
		next = encodeCursor(rows[limit-1].CreatedAt)
		rows = rows[:limit]
	}
	dtos, err := s.render(ctx, rows, actor)
	if err != nil {
		return Page{}, err
	}
	return Page{Posts: dtos, NextCursor: next}, nil
}

// render attaches media, reactions, authors and communities to a page of
// posts — four batched queries, not four per row.
func (s *Service) render(ctx context.Context, rows []Post, actor ActorMeta) ([]DTO, error) {
	if len(rows) == 0 {
		return []DTO{}, nil
	}
	postIDs := make([]uuid.UUID, 0, len(rows))
	authorIDs := make([]uuid.UUID, 0, len(rows))
	communityIDs := make([]uuid.UUID, 0, len(rows))
	for _, p := range rows {
		postIDs = append(postIDs, p.ID)
		authorIDs = append(authorIDs, p.AuthorID)
		communityIDs = append(communityIDs, p.CommunityID)
	}

	media, err := s.repo.MediaFor(ctx, s.pool, postIDs)
	if err != nil {
		return nil, err
	}
	reactions, err := s.repo.ReactionsFor(ctx, s.pool, postIDs, actor.UserID)
	if err != nil {
		return nil, err
	}
	communities, err := s.communities.ResolveMany(ctx, communityIDs, actor)
	if err != nil {
		return nil, err
	}
	cards := map[uuid.UUID]AuthorCard{}
	if s.profiles != nil {
		if cards, err = s.profiles.Cards(ctx, authorIDs); err != nil {
			return nil, err
		}
	}

	out := make([]DTO, 0, len(rows))
	for _, p := range rows {
		community := communities[p.CommunityID]
		dto := DTO{
			ID:        p.ID,
			Text:      p.Text,
			CreatedAt: p.CreatedAt,
			Author:    cards[p.AuthorID],
			Community: CommunityCard{
				ID:         community.ID,
				Name:       community.Name,
				AvatarURL:  community.AvatarURL,
				IsMember:   community.IsMember,
				JoinPolicy: community.JoinPolicy,
			},
			Media:     mediaDTOs(media[p.ID]),
			Reactions: reactions[p.ID],
			CanDelete: p.AuthorID == actor.UserID || community.CanPost,
		}
		if p.UpdatedAt.After(p.CreatedAt.Add(time.Second)) {
			edited := p.UpdatedAt
			dto.EditedAt = &edited
		}
		if dto.Reactions == nil {
			dto.Reactions = []ReactionSummary{}
		}
		out = append(out, dto)
	}
	return out, nil
}

func mediaDTOs(media []Media) []MediaDTO {
	out := make([]MediaDTO, 0, len(media))
	for _, m := range media {
		out = append(out, MediaDTO{
			ID:       m.ID,
			Kind:     m.Kind,
			FileName: m.FileName,
			MimeType: m.MimeType,
			Size:     m.SizeBytes,
			URL:      "/api/v1/posts/media/" + m.ID.String(),
			Position: m.Position,
		})
	}
	return out
}

// Cursors are opaque on purpose: what they encode is an implementation detail
// that differs between the two sorts (a timestamp and an offset).
func encodeCursor(t time.Time) string {
	return base64.RawURLEncoding.EncodeToString([]byte("t:" + t.UTC().Format(time.RFC3339Nano)))
}

func decodeCursor(cursor string) (time.Time, error) {
	if cursor == "" {
		return time.Time{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || !strings.HasPrefix(string(raw), "t:") {
		return time.Time{}, ErrNotFound
	}
	t, err := time.Parse(time.RFC3339Nano, strings.TrimPrefix(string(raw), "t:"))
	if err != nil {
		return time.Time{}, ErrNotFound
	}
	return t, nil
}

func encodeOffset(n int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("o:%d", n)))
}

func decodeOffset(cursor string) int {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	var n int
	if _, err := fmt.Sscanf(string(raw), "o:%d", &n); err != nil || n < 0 {
		return 0
	}
	return n
}
