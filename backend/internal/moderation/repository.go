package moderation

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

// Repository is the persistence of sanctions and the group columns they move.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

const sanctionColumns = `id, group_id, kind, reason, issued_by, issued_at, expires_at, revoked_at, revoked_by, revoke_note`

func scanSanction(row pgx.Row, s *Sanction) error {
	return row.Scan(&s.ID, &s.GroupID, &s.Kind, &s.Reason, &s.IssuedBy, &s.IssuedAt,
		&s.ExpiresAt, &s.RevokedAt, &s.RevokedBy, &s.RevokeNote)
}

func collectSanctions(rows pgx.Rows) ([]Sanction, error) {
	defer rows.Close()
	out := []Sanction{}
	for rows.Next() {
		var s Sanction
		if err := scanSanction(rows, &s); err != nil {
			return nil, fmt.Errorf("moderation: scan sanction: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// lockGroup reads a group and holds its row until the transaction ends. Two
// warnings issued at once must not both see "two so far" and both stop short
// of deleting.
func (r *Repository) lockGroup(ctx context.Context, q db.DBTX, id uuid.UUID) (*groupRow, error) {
	var g groupRow
	err := q.QueryRow(ctx, `
		SELECT id, name, kind, created_by, deleted_at FROM groups WHERE id = $1 FOR UPDATE`, id).
		Scan(&g.ID, &g.Name, &g.Kind, &g.CreatedBy, &g.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("moderation: lock group: %w", err)
	}
	return &g, nil
}

func (r *Repository) getGroup(ctx context.Context, q db.DBTX, id uuid.UUID) (*groupRow, error) {
	var g groupRow
	err := q.QueryRow(ctx, `SELECT id, name, kind, created_by, deleted_at FROM groups WHERE id = $1`, id).
		Scan(&g.ID, &g.Name, &g.Kind, &g.CreatedBy, &g.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("moderation: get group: %w", err)
	}
	return &g, nil
}

func (r *Repository) insert(ctx context.Context, q db.DBTX, s *Sanction) error {
	err := q.QueryRow(ctx, `
		INSERT INTO group_sanctions (group_id, kind, reason, issued_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, issued_at`,
		s.GroupID, s.Kind, s.Reason, s.IssuedBy, s.ExpiresAt).Scan(&s.ID, &s.IssuedAt)
	if err != nil {
		return fmt.Errorf("moderation: insert sanction: %w", err)
	}
	return nil
}

func (r *Repository) get(ctx context.Context, q db.DBTX, id uuid.UUID) (*Sanction, error) {
	var s Sanction
	err := scanSanction(q.QueryRow(ctx, `SELECT `+sanctionColumns+` FROM group_sanctions WHERE id = $1`, id), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("moderation: get sanction: %w", err)
	}
	return &s, nil
}

// activeWarns lists the group's live warnings, oldest first.
func (r *Repository) activeWarns(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]Sanction, error) {
	rows, err := q.Query(ctx, `
		SELECT `+sanctionColumns+` FROM group_sanctions
		WHERE group_id = $1 AND kind = 'warn' AND revoked_at IS NULL
		ORDER BY issued_at, id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("moderation: active warns: %w", err)
	}
	return collectSanctions(rows)
}

// liveMute returns the group's live mute, or nil.
func (r *Repository) liveMute(ctx context.Context, q db.DBTX, groupID uuid.UUID) (*Sanction, error) {
	var s Sanction
	err := scanSanction(q.QueryRow(ctx, `
		SELECT `+sanctionColumns+` FROM group_sanctions
		WHERE group_id = $1 AND kind = 'mute' AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > now())
		ORDER BY issued_at DESC LIMIT 1`, groupID), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("moderation: live mute: %w", err)
	}
	return &s, nil
}

// revoke ends a sanction that has not been revoked yet; false when it already was.
func (r *Repository) revoke(ctx context.Context, q db.DBTX, id, by uuid.UUID, note string, at time.Time) (bool, error) {
	var notePtr *string
	if strings.TrimSpace(note) != "" {
		notePtr = &note
	}
	tag, err := q.Exec(ctx, `
		UPDATE group_sanctions SET revoked_at = $2, revoked_by = $3, revoke_note = $4
		WHERE id = $1 AND revoked_at IS NULL`, id, at, by, notePtr)
	if err != nil {
		return false, fmt.Errorf("moderation: revoke: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// revokeLiveDeletions closes the 'delete' sanctions a restore undoes.
func (r *Repository) revokeLiveDeletions(ctx context.Context, q db.DBTX, groupID, by uuid.UUID, note string, at time.Time) error {
	_, err := q.Exec(ctx, `
		UPDATE group_sanctions SET revoked_at = $2, revoked_by = $3, revoke_note = $4
		WHERE group_id = $1 AND kind = 'delete' AND revoked_at IS NULL`, groupID, at, by, note)
	if err != nil {
		return fmt.Errorf("moderation: revoke deletions: %w", err)
	}
	return nil
}

func (r *Repository) setDeleted(ctx context.Context, q db.DBTX, groupID uuid.UUID, by *uuid.UUID, at *time.Time) error {
	_, err := q.Exec(ctx, `UPDATE groups SET deleted_at = $2, deleted_by = $3 WHERE id = $1`, groupID, at, by)
	if err != nil {
		return fmt.Errorf("moderation: set deleted: %w", err)
	}
	return nil
}

func (r *Repository) history(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]Sanction, error) {
	rows, err := q.Query(ctx, `
		SELECT `+sanctionColumns+` FROM group_sanctions
		WHERE group_id = $1 ORDER BY issued_at DESC, id DESC`, groupID)
	if err != nil {
		return nil, fmt.Errorf("moderation: history: %w", err)
	}
	return collectSanctions(rows)
}

// recipients are the people a sanction is addressed to: the founder and every
// member with the editor role or above.
func (r *Repository) recipients(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, `
		SELECT created_by FROM groups WHERE id = $1
		UNION
		SELECT user_id FROM group_members
		WHERE group_id = $1 AND role_in_group IN ('owner', 'editor', 'moderator')`, groupID)
	if err != nil {
		return nil, fmt.Errorf("moderation: recipients: %w", err)
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("moderation: scan recipient: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *Repository) listGroups(ctx context.Context, q db.DBTX, query string, limit int) ([]GroupSummary, error) {
	rows, err := q.Query(ctx, `
		SELECT g.id, g.name, g.kind, g.avatar_url, g.is_public, g.verified_at,
		       (SELECT count(*) FROM group_members m WHERE m.group_id = g.id)::int,
		       (SELECT count(*) FROM group_sanctions s
		         WHERE s.group_id = g.id AND s.kind = 'warn' AND s.revoked_at IS NULL)::int,
		       mute.id IS NOT NULL, mute.expires_at
		FROM groups g
		LEFT JOIN LATERAL (
		    SELECT s.id, s.expires_at FROM group_sanctions s
		    WHERE s.group_id = g.id AND s.kind = 'mute' AND s.revoked_at IS NULL
		      AND (s.expires_at IS NULL OR s.expires_at > now())
		    ORDER BY s.issued_at DESC LIMIT 1
		) mute ON true
		WHERE g.deleted_at IS NULL
		  AND ($1 = '' OR strpos(lower(g.name), lower($1)) > 0)
		ORDER BY g.created_at DESC, g.id DESC
		LIMIT $2`, strings.TrimSpace(query), limit)
	if err != nil {
		return nil, fmt.Errorf("moderation: list groups: %w", err)
	}
	defer rows.Close()
	out := []GroupSummary{}
	for rows.Next() {
		g := GroupSummary{WarnLimit: WarnLimit}
		if err := rows.Scan(&g.ID, &g.Name, &g.Kind, &g.AvatarURL, &g.IsPublic, &g.VerifiedAt,
			&g.MemberCount, &g.ActiveWarns, &g.Muted, &g.MutedUntil); err != nil {
			return nil, fmt.Errorf("moderation: scan group: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) listDeleted(ctx context.Context, q db.DBTX) ([]DeletedGroup, error) {
	rows, err := q.Query(ctx, `
		SELECT g.id, g.name, g.kind, g.avatar_url, g.deleted_at,
		       COALESCE((SELECT s.reason FROM group_sanctions s
		                  WHERE s.group_id = g.id AND s.kind = 'delete' AND s.revoked_at IS NULL
		                  ORDER BY s.issued_at DESC LIMIT 1), '')
		FROM groups g
		WHERE g.deleted_at IS NOT NULL
		ORDER BY g.deleted_at`)
	if err != nil {
		return nil, fmt.Errorf("moderation: list deleted: %w", err)
	}
	defer rows.Close()
	out := []DeletedGroup{}
	for rows.Next() {
		var d DeletedGroup
		if err := rows.Scan(&d.ID, &d.Name, &d.Kind, &d.AvatarURL, &d.DeletedAt, &d.DeleteReason); err != nil {
			return nil, fmt.Errorf("moderation: scan deleted: %w", err)
		}
		d.PurgeAt = d.DeletedAt.Add(RestoreWindow)
		out = append(out, d)
	}
	return out, rows.Err()
}

// expired lists deleted groups whose restore window closed before cutoff.
func (r *Repository) expired(ctx context.Context, q db.DBTX, cutoff time.Time) ([]groupRow, error) {
	rows, err := q.Query(ctx, `
		SELECT id, name, kind, created_by, deleted_at FROM groups
		WHERE deleted_at IS NOT NULL AND deleted_at < $1`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("moderation: expired: %w", err)
	}
	defer rows.Close()
	var out []groupRow
	for rows.Next() {
		var g groupRow
		if err := rows.Scan(&g.ID, &g.Name, &g.Kind, &g.CreatedBy, &g.DeletedAt); err != nil {
			return nil, fmt.Errorf("moderation: scan expired: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// purge removes a group for good: its messages (whose polymorphic chat_id has
// no foreign key) and the group, which cascades to members, posts, boards,
// calendar and sanctions. Only a group still deleted is touched.
func (r *Repository) purge(ctx context.Context, q db.DBTX, groupID uuid.UUID) (bool, error) {
	if _, err := q.Exec(ctx, `DELETE FROM messages WHERE chat_type = 'group' AND chat_id = $1`, groupID); err != nil {
		return false, fmt.Errorf("moderation: purge messages: %w", err)
	}
	tag, err := q.Exec(ctx, `DELETE FROM groups WHERE id = $1 AND deleted_at IS NOT NULL`, groupID)
	if err != nil {
		return false, fmt.Errorf("moderation: purge group: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
