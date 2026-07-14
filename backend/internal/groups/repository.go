package groups

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"kisy-backend/internal/platform/db"
)

const pgUniqueViolation = "23505"

// Repository is the persistence port for groups and membership.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, g *Group) error
	GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Group, error)
	// ListVisible returns non-archived groups the given clearance may see
	// (min_role_level >= actorLevel), newest first.
	ListVisible(ctx context.Context, q db.DBTX, actorLevel int) ([]Group, error)
	AddMember(ctx context.Context, q db.DBTX, m *Member) error
	IsMember(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (bool, error)
	ListMemberIDs(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]uuid.UUID, error)
	Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error
	// DeleteGroupMessages removes the group's messages, whose polymorphic
	// chat_id has no cascading foreign key.
	DeleteGroupMessages(ctx context.Context, q db.DBTX, groupID uuid.UUID) error
	// SetAvatarURL points the group's avatar_url at a (versioned) URL.
	SetAvatarURL(ctx context.Context, q db.DBTX, id uuid.UUID, url string) error
	// SetMinRoleLevel changes the group's minimum clearance (its "level").
	SetMinRoleLevel(ctx context.Context, q db.DBTX, id uuid.UUID, level int) error
	// SetPolicies updates a group's join_policy and post_policy.
	SetPolicies(ctx context.Context, q db.DBTX, id uuid.UUID, joinPolicy, postPolicy string) error
	// MemberRole returns a member's role_in_group; ok=false if not a member.
	MemberRole(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (role string, ok bool, err error)
	// ListMembersWithRoles returns each member's id and role_in_group.
	ListMembersWithRoles(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]MemberRole, error)
	// SetMemberRole changes an existing member's role.
	SetMemberRole(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID, role string) error
	// ListDirectory returns non-archived groups the clearance may see and is
	// NOT a member of, newest first, each with the actor's pending-request
	// status (empty when none).
	ListDirectory(ctx context.Context, q db.DBTX, actorID uuid.UUID, actorLevel int) ([]DirectoryEntry, error)
	// UpsertJoinRequest creates a pending request, or returns the existing one
	// if the user already has a pending request for the group.
	UpsertJoinRequest(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (*JoinRequest, error)
	// PendingRequest returns the user's pending request for a group, if any.
	PendingRequest(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (*JoinRequest, error)
	// ListPendingRequestUserIDs returns the user IDs with a pending request.
	ListPendingRequestUserIDs(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]uuid.UUID, error)
	// DecidePendingRequest marks the pending request approved/rejected. Returns
	// ErrRequestNotFound if there is no pending request.
	DecidePendingRequest(ctx context.Context, q db.DBTX, groupID, userID, decidedBy uuid.UUID, status string) error
}

// DirectoryEntry is a group plus the actor's pending-request status.
type DirectoryEntry struct {
	Group
	RequestStatus string
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const groupColumns = `id, name, description, avatar_url, min_role_level, join_policy, post_policy, created_by, is_archived, created_at, updated_at`

func scanGroupInto(row pgx.Row, g *Group) error {
	return row.Scan(&g.ID, &g.Name, &g.Description, &g.AvatarURL, &g.MinRoleLevel,
		&g.JoinPolicy, &g.PostPolicy, &g.CreatedBy, &g.IsArchived, &g.CreatedAt, &g.UpdatedAt)
}

func scanGroup(row pgx.Row) (*Group, error) {
	var g Group
	err := scanGroupInto(row, &g)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("groups: scan: %w", err)
	}
	return &g, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, g *Group) error {
	err := q.QueryRow(ctx, `
		INSERT INTO groups (name, description, avatar_url, min_role_level, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, join_policy, post_policy, is_archived, created_at, updated_at`,
		g.Name, g.Description, g.AvatarURL, g.MinRoleLevel, g.CreatedBy,
	).Scan(&g.ID, &g.JoinPolicy, &g.PostPolicy, &g.IsArchived, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("groups: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Group, error) {
	return scanGroup(q.QueryRow(ctx, `SELECT `+groupColumns+` FROM groups WHERE id = $1`, id))
}

func (r *PostgresRepository) ListVisible(ctx context.Context, q db.DBTX, actorLevel int) ([]Group, error) {
	rows, err := q.Query(ctx, `
		SELECT `+groupColumns+`
		FROM groups
		WHERE is_archived = false AND min_role_level >= $1
		ORDER BY created_at DESC, id DESC`, actorLevel)
	if err != nil {
		return nil, fmt.Errorf("groups: list visible: %w", err)
	}
	defer rows.Close()

	var out []Group
	for rows.Next() {
		var g Group
		if err := scanGroupInto(rows, &g); err != nil {
			return nil, fmt.Errorf("groups: scan row: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) SetPolicies(ctx context.Context, q db.DBTX, id uuid.UUID, joinPolicy, postPolicy string) error {
	tag, err := q.Exec(ctx, `UPDATE groups SET join_policy = $2, post_policy = $3, updated_at = now() WHERE id = $1`,
		id, joinPolicy, postPolicy)
	if err != nil {
		return fmt.Errorf("groups: set policies: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) MemberRole(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (string, bool, error) {
	var role string
	err := q.QueryRow(ctx, `SELECT role_in_group FROM group_members WHERE group_id = $1 AND user_id = $2`, groupID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("groups: member role: %w", err)
	}
	return role, true, nil
}

// MemberRole pairs a member id with its in-group role.
type MemberRole struct {
	UserID uuid.UUID
	Role   string
}

func (r *PostgresRepository) ListMembersWithRoles(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]MemberRole, error) {
	rows, err := q.Query(ctx, `SELECT user_id, role_in_group FROM group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, fmt.Errorf("groups: list members with roles: %w", err)
	}
	defer rows.Close()
	var out []MemberRole
	for rows.Next() {
		var m MemberRole
		if err := rows.Scan(&m.UserID, &m.Role); err != nil {
			return nil, fmt.Errorf("groups: scan member role: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) SetMemberRole(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID, role string) error {
	tag, err := q.Exec(ctx, `UPDATE group_members SET role_in_group = $3 WHERE group_id = $1 AND user_id = $2`, groupID, userID, role)
	if err != nil {
		return fmt.Errorf("groups: set member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}
	return nil
}

func (r *PostgresRepository) ListDirectory(ctx context.Context, q db.DBTX, actorID uuid.UUID, actorLevel int) ([]DirectoryEntry, error) {
	rows, err := q.Query(ctx, `
		SELECT `+prefixedGroupColumns("g")+`,
		       COALESCE(jr.status, '') AS request_status
		FROM groups g
		LEFT JOIN group_join_requests jr
		       ON jr.group_id = g.id AND jr.user_id = $1 AND jr.status = 'pending'
		WHERE g.is_archived = false
		  AND g.min_role_level >= $2
		  AND NOT EXISTS (SELECT 1 FROM group_members m WHERE m.group_id = g.id AND m.user_id = $1)
		ORDER BY g.created_at DESC, g.id DESC`, actorID, actorLevel)
	if err != nil {
		return nil, fmt.Errorf("groups: list directory: %w", err)
	}
	defer rows.Close()

	var out []DirectoryEntry
	for rows.Next() {
		var e DirectoryEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.AvatarURL, &e.MinRoleLevel,
			&e.JoinPolicy, &e.PostPolicy, &e.CreatedBy, &e.IsArchived, &e.CreatedAt, &e.UpdatedAt,
			&e.RequestStatus); err != nil {
			return nil, fmt.Errorf("groups: scan directory row: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) scanRequest(row pgx.Row) (*JoinRequest, error) {
	var jr JoinRequest
	err := row.Scan(&jr.ID, &jr.GroupID, &jr.UserID, &jr.Status, &jr.RequestedAt, &jr.DecidedBy, &jr.DecidedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRequestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("groups: scan request: %w", err)
	}
	return &jr, nil
}

const joinRequestColumns = `id, group_id, user_id, status, requested_at, decided_by, decided_at`

func (r *PostgresRepository) UpsertJoinRequest(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (*JoinRequest, error) {
	// Insert a pending request; if one already exists (partial unique index on
	// pending), return it unchanged.
	jr, err := r.scanRequest(q.QueryRow(ctx, `
		INSERT INTO group_join_requests (group_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (group_id, user_id) WHERE status = 'pending' DO NOTHING
		RETURNING `+joinRequestColumns, groupID, userID))
	if errors.Is(err, ErrRequestNotFound) {
		// Conflict → the pending row already exists; fetch it.
		return r.PendingRequest(ctx, q, groupID, userID)
	}
	return jr, err
}

func (r *PostgresRepository) PendingRequest(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (*JoinRequest, error) {
	return r.scanRequest(q.QueryRow(ctx, `
		SELECT `+joinRequestColumns+` FROM group_join_requests
		WHERE group_id = $1 AND user_id = $2 AND status = 'pending'`, groupID, userID))
}

func (r *PostgresRepository) ListPendingRequestUserIDs(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, `
		SELECT user_id FROM group_join_requests
		WHERE group_id = $1 AND status = 'pending'
		ORDER BY requested_at ASC`, groupID)
	if err != nil {
		return nil, fmt.Errorf("groups: list pending requests: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("groups: scan pending id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) DecidePendingRequest(ctx context.Context, q db.DBTX, groupID, userID, decidedBy uuid.UUID, status string) error {
	tag, err := q.Exec(ctx, `
		UPDATE group_join_requests
		SET status = $4, decided_by = $3, decided_at = now()
		WHERE group_id = $1 AND user_id = $2 AND status = 'pending'`,
		groupID, userID, decidedBy, status)
	if err != nil {
		return fmt.Errorf("groups: decide request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRequestNotFound
	}
	return nil
}

// prefixedGroupColumns returns groupColumns with a table alias prefix.
func prefixedGroupColumns(alias string) string {
	return alias + ".id, " + alias + ".name, " + alias + ".description, " + alias + ".avatar_url, " +
		alias + ".min_role_level, " + alias + ".join_policy, " + alias + ".post_policy, " +
		alias + ".created_by, " + alias + ".is_archived, " + alias + ".created_at, " + alias + ".updated_at"
}

func (r *PostgresRepository) AddMember(ctx context.Context, q db.DBTX, m *Member) error {
	_, err := q.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id, role_in_group)
		VALUES ($1, $2, $3)`,
		m.GroupID, m.UserID, m.RoleInGroup,
	)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return ErrAlreadyMember
	}
	if err != nil {
		return fmt.Errorf("groups: add member: %w", err)
	}
	return nil
}

func (r *PostgresRepository) IsMember(ctx context.Context, q db.DBTX, groupID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`,
		groupID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("groups: is member: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("groups: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteGroupMessages(ctx context.Context, q db.DBTX, groupID uuid.UUID) error {
	if _, err := q.Exec(ctx, `DELETE FROM messages WHERE chat_type = 'group' AND chat_id = $1`, groupID); err != nil {
		return fmt.Errorf("groups: delete group messages: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetAvatarURL(ctx context.Context, q db.DBTX, id uuid.UUID, url string) error {
	tag, err := q.Exec(ctx, `UPDATE groups SET avatar_url = $2, updated_at = now() WHERE id = $1`, id, url)
	if err != nil {
		return fmt.Errorf("groups: set avatar url: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetMinRoleLevel(ctx context.Context, q db.DBTX, id uuid.UUID, level int) error {
	tag, err := q.Exec(ctx, `UPDATE groups SET min_role_level = $2, updated_at = now() WHERE id = $1`, id, level)
	if err != nil {
		return fmt.Errorf("groups: set min role level: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListMemberIDs(ctx context.Context, q db.DBTX, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, `SELECT user_id FROM group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, fmt.Errorf("groups: list member ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("groups: scan member id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
