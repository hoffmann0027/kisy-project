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
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const groupColumns = `id, name, description, avatar_url, min_role_level, created_by, is_archived, created_at, updated_at`

func scanGroup(row pgx.Row) (*Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.Description, &g.AvatarURL, &g.MinRoleLevel,
		&g.CreatedBy, &g.IsArchived, &g.CreatedAt, &g.UpdatedAt)
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
		RETURNING id, is_archived, created_at, updated_at`,
		g.Name, g.Description, g.AvatarURL, g.MinRoleLevel, g.CreatedBy,
	).Scan(&g.ID, &g.IsArchived, &g.CreatedAt, &g.UpdatedAt)
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
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.AvatarURL, &g.MinRoleLevel,
			&g.CreatedBy, &g.IsArchived, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("groups: scan row: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
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
