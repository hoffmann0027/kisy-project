package boards

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// Repository is the persistence port for boards, columns and cards.
type Repository interface {
	CreateBoard(ctx context.Context, q db.DBTX, b *Board) error
	GetBoardByGroup(ctx context.Context, q db.DBTX, groupID uuid.UUID) (*Board, error)
	GetBoardByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Board, error)

	CreateColumn(ctx context.Context, q db.DBTX, c *Column) error
	ListColumns(ctx context.Context, q db.DBTX, boardID uuid.UUID) ([]Column, error)
	GetColumn(ctx context.Context, q db.DBTX, id uuid.UUID) (*Column, error)
	RenameColumn(ctx context.Context, q db.DBTX, id uuid.UUID, title string) error
	DeleteColumn(ctx context.Context, q db.DBTX, id uuid.UUID) error
	MaxColumnPosition(ctx context.Context, q db.DBTX, boardID uuid.UUID) (int, error)

	CreateCard(ctx context.Context, q db.DBTX, c *Card) error
	ListCards(ctx context.Context, q db.DBTX, boardID uuid.UUID) ([]Card, error)
	GetCard(ctx context.Context, q db.DBTX, id uuid.UUID) (*Card, error)
	UpdateCard(ctx context.Context, q db.DBTX, c *Card) error
	DeleteCard(ctx context.Context, q db.DBTX, id uuid.UUID) error
	MaxCardPosition(ctx context.Context, q db.DBTX, columnID uuid.UUID) (int, error)
	// ListCardIDsInColumn returns card ids ordered by position, for
	// re-sequencing during a move.
	ListCardIDsInColumn(ctx context.Context, q db.DBTX, columnID uuid.UUID) ([]uuid.UUID, error)
	SetCardPosition(ctx context.Context, q db.DBTX, cardID, columnID uuid.UUID, position int) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

// --- boards ---

func (r *PostgresRepository) CreateBoard(ctx context.Context, q db.DBTX, b *Board) error {
	err := q.QueryRow(ctx, `
		INSERT INTO boards (group_id, title, created_by) VALUES ($1, $2, $3)
		RETURNING id, created_at`, b.GroupID, b.Title, b.CreatedBy,
	).Scan(&b.ID, &b.CreatedAt)
	if err != nil {
		return fmt.Errorf("boards: create board: %w", err)
	}
	return nil
}

func scanBoard(row pgx.Row) (*Board, error) {
	var b Board
	err := row.Scan(&b.ID, &b.GroupID, &b.Title, &b.CreatedBy, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("boards: scan board: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) GetBoardByGroup(ctx context.Context, q db.DBTX, groupID uuid.UUID) (*Board, error) {
	return scanBoard(q.QueryRow(ctx, `SELECT id, group_id, title, created_by, created_at FROM boards WHERE group_id = $1`, groupID))
}

func (r *PostgresRepository) GetBoardByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Board, error) {
	return scanBoard(q.QueryRow(ctx, `SELECT id, group_id, title, created_by, created_at FROM boards WHERE id = $1`, id))
}

// --- columns ---

func (r *PostgresRepository) CreateColumn(ctx context.Context, q db.DBTX, c *Column) error {
	err := q.QueryRow(ctx, `
		INSERT INTO board_columns (board_id, title, position) VALUES ($1, $2, $3) RETURNING id`,
		c.BoardID, c.Title, c.Position,
	).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("boards: create column: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListColumns(ctx context.Context, q db.DBTX, boardID uuid.UUID) ([]Column, error) {
	rows, err := q.Query(ctx, `SELECT id, board_id, title, position FROM board_columns WHERE board_id = $1 ORDER BY position ASC`, boardID)
	if err != nil {
		return nil, fmt.Errorf("boards: list columns: %w", err)
	}
	defer rows.Close()
	var out []Column
	for rows.Next() {
		var c Column
		if err := rows.Scan(&c.ID, &c.BoardID, &c.Title, &c.Position); err != nil {
			return nil, fmt.Errorf("boards: scan column: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetColumn(ctx context.Context, q db.DBTX, id uuid.UUID) (*Column, error) {
	var c Column
	err := q.QueryRow(ctx, `SELECT id, board_id, title, position FROM board_columns WHERE id = $1`, id).
		Scan(&c.ID, &c.BoardID, &c.Title, &c.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("boards: get column: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) RenameColumn(ctx context.Context, q db.DBTX, id uuid.UUID, title string) error {
	tag, err := q.Exec(ctx, `UPDATE board_columns SET title = $2 WHERE id = $1`, id, title)
	if err != nil {
		return fmt.Errorf("boards: rename column: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteColumn(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM board_columns WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("boards: delete column: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) MaxColumnPosition(ctx context.Context, q db.DBTX, boardID uuid.UUID) (int, error) {
	var pos *int
	if err := q.QueryRow(ctx, `SELECT max(position) FROM board_columns WHERE board_id = $1`, boardID).Scan(&pos); err != nil {
		return 0, fmt.Errorf("boards: max column position: %w", err)
	}
	if pos == nil {
		return -1, nil
	}
	return *pos, nil
}

// --- cards ---

const cardColumns = `id, board_id, column_id, title, description, position, assignee_id, label, due_date, created_by, created_at`

func scanCardRow(row pgx.Row) (*Card, error) {
	var c Card
	err := row.Scan(&c.ID, &c.BoardID, &c.ColumnID, &c.Title, &c.Description, &c.Position,
		&c.AssigneeID, &c.Label, &c.DueDate, &c.CreatedBy, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("boards: scan card: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) CreateCard(ctx context.Context, q db.DBTX, c *Card) error {
	err := q.QueryRow(ctx, `
		INSERT INTO board_cards (board_id, column_id, title, description, position, assignee_id, label, due_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		c.BoardID, c.ColumnID, c.Title, c.Description, c.Position, c.AssigneeID, c.Label, c.DueDate, c.CreatedBy,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("boards: create card: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListCards(ctx context.Context, q db.DBTX, boardID uuid.UUID) ([]Card, error) {
	rows, err := q.Query(ctx, `SELECT `+cardColumns+` FROM board_cards WHERE board_id = $1 ORDER BY column_id, position ASC`, boardID)
	if err != nil {
		return nil, fmt.Errorf("boards: list cards: %w", err)
	}
	defer rows.Close()
	var out []Card
	for rows.Next() {
		var c Card
		if err := rows.Scan(&c.ID, &c.BoardID, &c.ColumnID, &c.Title, &c.Description, &c.Position,
			&c.AssigneeID, &c.Label, &c.DueDate, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("boards: scan card row: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetCard(ctx context.Context, q db.DBTX, id uuid.UUID) (*Card, error) {
	return scanCardRow(q.QueryRow(ctx, `SELECT `+cardColumns+` FROM board_cards WHERE id = $1`, id))
}

func (r *PostgresRepository) UpdateCard(ctx context.Context, q db.DBTX, c *Card) error {
	tag, err := q.Exec(ctx, `
		UPDATE board_cards SET title = $2, description = $3, assignee_id = $4, label = $5, due_date = $6
		WHERE id = $1`,
		c.ID, c.Title, c.Description, c.AssigneeID, c.Label, c.DueDate)
	if err != nil {
		return fmt.Errorf("boards: update card: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteCard(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM board_cards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("boards: delete card: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) MaxCardPosition(ctx context.Context, q db.DBTX, columnID uuid.UUID) (int, error) {
	var pos *int
	if err := q.QueryRow(ctx, `SELECT max(position) FROM board_cards WHERE column_id = $1`, columnID).Scan(&pos); err != nil {
		return 0, fmt.Errorf("boards: max card position: %w", err)
	}
	if pos == nil {
		return -1, nil
	}
	return *pos, nil
}

func (r *PostgresRepository) ListCardIDsInColumn(ctx context.Context, q db.DBTX, columnID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, `SELECT id FROM board_cards WHERE column_id = $1 ORDER BY position ASC`, columnID)
	if err != nil {
		return nil, fmt.Errorf("boards: list card ids: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("boards: scan card id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) SetCardPosition(ctx context.Context, q db.DBTX, cardID, columnID uuid.UUID, position int) error {
	_, err := q.Exec(ctx, `UPDATE board_cards SET column_id = $2, position = $3 WHERE id = $1`, cardID, columnID, position)
	if err != nil {
		return fmt.Errorf("boards: set card position: %w", err)
	}
	return nil
}
