package posts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// Repository is the persistence port for posts, their media and their
// reactions.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, p *Post) error
	AddMedia(ctx context.Context, q db.DBTX, postID uuid.UUID, m []Media) error
	Get(ctx context.Context, q db.DBTX, id uuid.UUID) (*Post, error)
	SoftDelete(ctx context.Context, q db.DBTX, id uuid.UUID, at time.Time) error

	// ListByCommunity returns one community's wall, newest first. `before` is
	// the created_at of the last post already shown (zero for the first page).
	ListByCommunity(ctx context.Context, q db.DBTX, communityID uuid.UUID, before time.Time, limit int) ([]Post, error)
	// ListNewest returns the chronological feed across the public communities
	// this viewer may see.
	ListNewest(ctx context.Context, q db.DBTX, viewerID uuid.UUID, viewerLevel int, before time.Time, limit int) ([]Post, error)
	// ByIDsVisible filters a ranked list of ids down to the ones this viewer
	// may actually see, preserving the given order.
	ByIDsVisible(ctx context.Context, q db.DBTX, ids []uuid.UUID, viewerID uuid.UUID, viewerLevel int) ([]Post, error)

	// ScoreInputs returns what the popularity formula needs for every post in
	// the scoring window (docs/spec/07-business-logic.md).
	ScoreInputs(ctx context.Context, q db.DBTX, since time.Time) ([]ScoreInput, error)

	MediaFor(ctx context.Context, q db.DBTX, postIDs []uuid.UUID) (map[uuid.UUID][]Media, error)
	ReactionsFor(ctx context.Context, q db.DBTX, postIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]ReactionSummary, error)
	AddReaction(ctx context.Context, q db.DBTX, postID, userID uuid.UUID, emoji string) error
	RemoveReaction(ctx context.Context, q db.DBTX, postID, userID uuid.UUID, emoji string) error

	MediaByID(ctx context.Context, q db.DBTX, id uuid.UUID) (Media, uuid.UUID, error)

	HideCommunity(ctx context.Context, q db.DBTX, userID, groupID uuid.UUID) error
	ShowCommunity(ctx context.Context, q db.DBTX, userID, groupID uuid.UUID) error
}

// ScoreInput is one post's contribution to the ranking: how old it is and how
// many DIFFERENT people reacted to it.
type ScoreInput struct {
	PostID   uuid.UUID
	Reactors int
	Age      time.Duration
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const postColumns = `id, community_id, author_id, text, created_at, updated_at`

// visibleCommunity is the rule for a post appearing in a shared feed, as
// opposed to on its own community's wall.
//
// Public is not the same as unrestricted: is_public and min_role_level are
// independent axes (migration 43), so a community can be public to read and
// still sit above someone's clearance. Leaving the threshold out here would
// leak exactly the posts the hierarchy exists to hide.
//
// Moderation (migration 48) takes a community out of every feed in two ways:
// deleted (restorable for 30 days, but gone meanwhile) and muted (its posts
// stay on its own wall, where its editors may keep publishing, and only the
// shared feed leaves them out). Both are decided here, in SQL, so no reader's
// client ever receives a muted post to hide.
const visibleCommunity = `
	g.kind = 'community' AND g.is_public = true AND g.is_archived = false
	AND g.deleted_at IS NULL
	AND NOT ` + mutedNow + `
	AND (g.min_role_level IS NULL OR ($2 BETWEEN 1 AND 10 AND g.min_role_level >= $2))
	AND NOT EXISTS (
		SELECT 1 FROM feed_hidden_communities h
		WHERE h.user_id = $1 AND h.group_id = g.id
	)`

// mutedNow is true while the community g has a live mute: not revoked, and
// either indefinite or not yet expired. Shared by the feed pages and the
// ranking, so a muted community is neither shown nor ranked.
const mutedNow = `EXISTS (
		SELECT 1 FROM group_sanctions s
		WHERE s.group_id = g.id AND s.kind = 'mute' AND s.revoked_at IS NULL
		  AND (s.expires_at IS NULL OR s.expires_at > now())
	)`

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, p *Post) error {
	err := q.QueryRow(ctx, `
		INSERT INTO posts (community_id, author_id, text)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`,
		p.CommunityID, p.AuthorID, p.Text,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("posts: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AddMedia(ctx context.Context, q db.DBTX, postID uuid.UUID, media []Media) error {
	for i := range media {
		m := media[i]
		_, err := q.Exec(ctx, `
			INSERT INTO post_media (post_id, kind, file_name, mime_type, size_bytes, storage_path, bytes, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			postID, m.Kind, m.FileName, m.MimeType, m.SizeBytes, m.StoragePath, m.Bytes, i)
		if err != nil {
			return fmt.Errorf("posts: add media: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) Get(ctx context.Context, q db.DBTX, id uuid.UUID) (*Post, error) {
	var p Post
	err := q.QueryRow(ctx, `SELECT `+postColumns+` FROM posts WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&p.ID, &p.CommunityID, &p.AuthorID, &p.Text, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("posts: get: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, q db.DBTX, id uuid.UUID, at time.Time) error {
	tag, err := q.Exec(ctx, `UPDATE posts SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`, id, at)
	if err != nil {
		return fmt.Errorf("posts: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanPosts(rows pgx.Rows) ([]Post, error) {
	defer rows.Close()
	var out []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.CommunityID, &p.AuthorID, &p.Text, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("posts: scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListByCommunity(
	ctx context.Context, q db.DBTX, communityID uuid.UUID, before time.Time, limit int,
) ([]Post, error) {
	rows, err := q.Query(ctx, `
		SELECT `+postColumns+` FROM posts
		WHERE community_id = $1 AND deleted_at IS NULL
		  AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3`, communityID, nullTime(before), limit)
	if err != nil {
		return nil, fmt.Errorf("posts: list by community: %w", err)
	}
	return scanPosts(rows)
}

func (r *PostgresRepository) ListNewest(
	ctx context.Context, q db.DBTX, viewerID uuid.UUID, viewerLevel int, before time.Time, limit int,
) ([]Post, error) {
	rows, err := q.Query(ctx, `
		SELECT `+prefixed(postColumns, "p")+` FROM posts p
		JOIN groups g ON g.id = p.community_id
		WHERE p.deleted_at IS NULL AND`+visibleCommunity+`
		  AND ($3::timestamptz IS NULL OR p.created_at < $3)
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $4`, viewerID, viewerLevel, nullTime(before), limit)
	if err != nil {
		return nil, fmt.Errorf("posts: feed: %w", err)
	}
	return scanPosts(rows)
}

func (r *PostgresRepository) ByIDsVisible(
	ctx context.Context, q db.DBTX, ids []uuid.UUID, viewerID uuid.UUID, viewerLevel int,
) ([]Post, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := q.Query(ctx, `
		SELECT `+prefixed(postColumns, "p")+` FROM posts p
		JOIN groups g ON g.id = p.community_id
		WHERE p.deleted_at IS NULL AND p.id = ANY($3) AND`+visibleCommunity,
		viewerID, viewerLevel, ids)
	if err != nil {
		return nil, fmt.Errorf("posts: by ids: %w", err)
	}
	found, err := scanPosts(rows)
	if err != nil {
		return nil, err
	}
	// Ranking order comes from Redis and must survive the visibility filter.
	byID := make(map[uuid.UUID]Post, len(found))
	for _, p := range found {
		byID[p.ID] = p
	}
	ordered := make([]Post, 0, len(found))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			ordered = append(ordered, p)
		}
	}
	return ordered, nil
}

func (r *PostgresRepository) ScoreInputs(ctx context.Context, q db.DBTX, since time.Time) ([]ScoreInput, error) {
	// COUNT(DISTINCT user_id) counts people, which is what the formula is
	// about (docs/spec/07-business-logic.md). Since migration 45 a person has
	// at most one reaction per post, so this equals COUNT(rx.id) — kept
	// DISTINCT so the ranking does not silently depend on that constraint.
	rows, err := q.Query(ctx, `
		SELECT p.id,
		       COUNT(DISTINCT rx.user_id) AS reactors,
		       EXTRACT(EPOCH FROM (now() - p.created_at)) AS age_seconds
		FROM posts p
		JOIN groups g ON g.id = p.community_id
		LEFT JOIN reactions rx ON rx.post_id = p.id
		WHERE p.deleted_at IS NULL
		  AND g.kind = 'community' AND g.is_public = true AND g.is_archived = false
		  AND g.deleted_at IS NULL AND NOT `+mutedNow+`
		  AND p.created_at >= $1
		GROUP BY p.id, p.created_at`, since)
	if err != nil {
		return nil, fmt.Errorf("posts: score inputs: %w", err)
	}
	defer rows.Close()

	var out []ScoreInput
	for rows.Next() {
		var in ScoreInput
		var ageSeconds float64
		if err := rows.Scan(&in.PostID, &in.Reactors, &ageSeconds); err != nil {
			return nil, fmt.Errorf("posts: scan score input: %w", err)
		}
		in.Age = time.Duration(ageSeconds) * time.Second
		out = append(out, in)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) MediaFor(
	ctx context.Context, q db.DBTX, postIDs []uuid.UUID,
) (map[uuid.UUID][]Media, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID][]Media{}, nil
	}
	rows, err := q.Query(ctx, `
		SELECT post_id, id, kind, file_name, mime_type, size_bytes, storage_path, position
		FROM post_media WHERE post_id = ANY($1) ORDER BY post_id, position`, postIDs)
	if err != nil {
		return nil, fmt.Errorf("posts: media: %w", err)
	}
	defer rows.Close()

	out := map[uuid.UUID][]Media{}
	for rows.Next() {
		var postID uuid.UUID
		var m Media
		if err := rows.Scan(&postID, &m.ID, &m.Kind, &m.FileName, &m.MimeType, &m.SizeBytes, &m.StoragePath, &m.Position); err != nil {
			return nil, fmt.Errorf("posts: scan media: %w", err)
		}
		out[postID] = append(out[postID], m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ReactionsFor(
	ctx context.Context, q db.DBTX, postIDs []uuid.UUID, viewerID uuid.UUID,
) (map[uuid.UUID][]ReactionSummary, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID][]ReactionSummary{}, nil
	}
	rows, err := q.Query(ctx, `
		SELECT post_id, emoji, COUNT(*)::int, bool_or(user_id = $2)
		FROM reactions
		WHERE post_id = ANY($1)
		GROUP BY post_id, emoji
		ORDER BY post_id, COUNT(*) DESC, emoji`, postIDs, viewerID)
	if err != nil {
		return nil, fmt.Errorf("posts: reactions: %w", err)
	}
	defer rows.Close()

	out := map[uuid.UUID][]ReactionSummary{}
	for rows.Next() {
		var postID uuid.UUID
		var s ReactionSummary
		if err := rows.Scan(&postID, &s.Emoji, &s.Count, &s.Mine); err != nil {
			return nil, fmt.Errorf("posts: scan reaction: %w", err)
		}
		out[postID] = append(out[postID], s)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) AddReaction(ctx context.Context, q db.DBTX, postID, userID uuid.UUID, emoji string) error {
	// One reaction per person per post (migration 45): a second emoji replaces
	// the first rather than joining it. One statement, so two quick taps cannot
	// interleave into a delete-then-double-insert.
	//
	// The conflict target names the partial index's predicate on purpose —
	// without the WHERE, Postgres does not match a partial index and rejects
	// the statement outright.
	_, err := q.Exec(ctx, `
		INSERT INTO reactions (post_id, user_id, emoji) VALUES ($1, $2, $3)
		ON CONFLICT (post_id, user_id) WHERE post_id IS NOT NULL
		DO UPDATE SET emoji = EXCLUDED.emoji, created_at = now()`,
		postID, userID, emoji)
	if err != nil {
		return fmt.Errorf("posts: add reaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RemoveReaction(ctx context.Context, q db.DBTX, postID, userID uuid.UUID, emoji string) error {
	_, err := q.Exec(ctx,
		`DELETE FROM reactions WHERE post_id = $1 AND user_id = $2 AND emoji = $3`, postID, userID, emoji)
	if err != nil {
		return fmt.Errorf("posts: remove reaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) HideCommunity(ctx context.Context, q db.DBTX, userID, groupID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO feed_hidden_communities (user_id, group_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, userID, groupID)
	if err != nil {
		return fmt.Errorf("posts: hide community: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ShowCommunity(ctx context.Context, q db.DBTX, userID, groupID uuid.UUID) error {
	_, err := q.Exec(ctx,
		`DELETE FROM feed_hidden_communities WHERE user_id = $1 AND group_id = $2`, userID, groupID)
	if err != nil {
		return fmt.Errorf("posts: show community: %w", err)
	}
	return nil
}

// nullTime turns the zero time into SQL NULL, which the queries read as "no
// cursor, start from the top".
func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// prefixed qualifies a bare column list with a table alias, so the column
// list is written once and reused in queries that join.
func prefixed(columns, alias string) string {
	parts := strings.Split(columns, ",")
	for i, c := range parts {
		parts[i] = alias + "." + strings.TrimSpace(c)
	}
	return strings.Join(parts, ", ")
}

// MediaByID returns one attachment and the post it belongs to.
func (r *PostgresRepository) MediaByID(ctx context.Context, q db.DBTX, id uuid.UUID) (Media, uuid.UUID, error) {
	var m Media
	var postID uuid.UUID
	err := q.QueryRow(ctx, `
		SELECT id, post_id, kind, file_name, mime_type, size_bytes, storage_path, bytes, position
		FROM post_media WHERE id = $1`, id).
		Scan(&m.ID, &postID, &m.Kind, &m.FileName, &m.MimeType, &m.SizeBytes, &m.StoragePath, &m.Bytes, &m.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return Media{}, uuid.Nil, ErrNotFound
	}
	if err != nil {
		return Media{}, uuid.Nil, fmt.Errorf("posts: media by id: %w", err)
	}
	return m, postID, nil
}
