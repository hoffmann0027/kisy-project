package calendar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// Repository is the persistence port for calendar events.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, e *Event) error
	GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Event, error)
	// ListByGroupInterval returns a group's events overlapping [from, to),
	// ordered by start time.
	ListByGroupInterval(ctx context.Context, q db.DBTX, groupID uuid.UUID, from, to time.Time) ([]Event, error)
	Update(ctx context.Context, q db.DBTX, id uuid.UUID, title string, startsAt time.Time, endsAt *time.Time, color string) error
	Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const eventColumns = `id, group_id, title, starts_at, ends_at, color, created_by, created_at, updated_at`

func scanEvent(row pgx.Row) (*Event, error) {
	var e Event
	err := row.Scan(&e.ID, &e.GroupID, &e.Title, &e.StartsAt, &e.EndsAt, &e.Color, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("calendar: scan: %w", err)
	}
	return &e, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, e *Event) error {
	err := q.QueryRow(ctx, `
		INSERT INTO calendar_events (group_id, title, starts_at, ends_at, color, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		e.GroupID, e.Title, e.StartsAt, e.EndsAt, e.Color, e.CreatedBy,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("calendar: create: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, q db.DBTX, id uuid.UUID) (*Event, error) {
	return scanEvent(q.QueryRow(ctx, `SELECT `+eventColumns+` FROM calendar_events WHERE id = $1`, id))
}

func (r *PostgresRepository) ListByGroupInterval(ctx context.Context, q db.DBTX, groupID uuid.UUID, from, to time.Time) ([]Event, error) {
	// An event overlaps [from, to) if it starts before `to` and (has no end or
	// ends at/after `from`). Covers multi-day events touching the window.
	rows, err := q.Query(ctx, `
		SELECT `+eventColumns+`
		FROM calendar_events
		WHERE group_id = $1
		  AND starts_at < $3
		  AND COALESCE(ends_at, starts_at) >= $2
		ORDER BY starts_at ASC`, groupID, from, to)
	if err != nil {
		return nil, fmt.Errorf("calendar: list interval: %w", err)
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.GroupID, &e.Title, &e.StartsAt, &e.EndsAt, &e.Color, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("calendar: scan row: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, q db.DBTX, id uuid.UUID, title string, startsAt time.Time, endsAt *time.Time, color string) error {
	tag, err := q.Exec(ctx, `
		UPDATE calendar_events
		SET title = $2, starts_at = $3, ends_at = $4, color = $5, updated_at = now()
		WHERE id = $1`, id, title, startsAt, endsAt, color)
	if err != nil {
		return fmt.Errorf("calendar: update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM calendar_events WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("calendar: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
