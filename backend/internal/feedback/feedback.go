// Package feedback owns the "Отзывы и предложения" board: any authenticated
// user may post a suggestion; only the CEO may delete one. Deletion authority
// is enforced here in the service, never merely hidden in the UI.
package feedback

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

const (
	maxBodyLen     = 2000
	defaultPageLen = 20
	maxPageLen     = 50
)

var (
	ErrNotFound  = errors.New("feedback: not found")
	ErrForbidden = errors.New("feedback: only the CEO may delete feedback")
	ErrEmpty     = errors.New("feedback: body is empty")
	ErrTooLong   = errors.New("feedback: body too long")
)

// Author is the public identity of a feedback poster, embedded in the DTO so
// the client can render it without a second lookup.
type Author struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	Username    string    `json:"username"`
	AvatarURL   *string   `json:"avatarUrl"`
	RoleLevel   int       `json:"roleLevel"`
}

// DTO is the API representation of one feedback entry.
type DTO struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	Author    Author    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
}

// Page is a cursor-paginated slice of feedback, newest first.
type Page struct {
	Items      []DTO   `json:"items"`
	NextCursor *string `json:"nextCursor"`
	HasMore    bool    `json:"hasMore"`
}

// Repository is the persistence port for feedback.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, authorID uuid.UUID, body string) (DTO, error)
	List(ctx context.Context, q db.DBTX, before *time.Time, limit int) ([]DTO, error)
	Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const selectColumns = `
	f.id, f.body, f.created_at,
	u.id, u.display_name, u.username, u.avatar_url, u.role_id`

func scan(row pgx.Row) (DTO, error) {
	var d DTO
	if err := row.Scan(&d.ID, &d.Body, &d.CreatedAt,
		&d.Author.ID, &d.Author.DisplayName, &d.Author.Username, &d.Author.AvatarURL, &d.Author.RoleLevel); err != nil {
		return DTO{}, err
	}
	return d, nil
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, authorID uuid.UUID, body string) (DTO, error) {
	row := q.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO feedback (author_id, body) VALUES ($1, $2)
			RETURNING id, body, created_at, author_id
		)
		SELECT `+selectColumns+`
		FROM inserted f JOIN users u ON u.id = f.author_id`,
		authorID, body)
	d, err := scan(row)
	if err != nil {
		return DTO{}, fmt.Errorf("feedback: create: %w", err)
	}
	return d, nil
}

func (r *PostgresRepository) List(ctx context.Context, q db.DBTX, before *time.Time, limit int) ([]DTO, error) {
	rows, err := q.Query(ctx, `
		SELECT `+selectColumns+`
		FROM feedback f JOIN users u ON u.id = f.author_id
		WHERE ($1::timestamptz IS NULL OR f.created_at < $1)
		ORDER BY f.created_at DESC, f.id DESC
		LIMIT $2`,
		before, limit)
	if err != nil {
		return nil, fmt.Errorf("feedback: list: %w", err)
	}
	defer rows.Close()

	var out []DTO
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("feedback: scan: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM feedback WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("feedback: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Actor identifies the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

type Service struct {
	pool *pgxpool.Pool
	repo Repository
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// Create validates and stores a feedback entry.
func (s *Service) Create(ctx context.Context, authorID uuid.UUID, body string) (DTO, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return DTO{}, ErrEmpty
	}
	if len([]rune(body)) > maxBodyLen {
		return DTO{}, ErrTooLong
	}
	return s.repo.Create(ctx, s.pool, authorID, body)
}

// List returns a page of feedback newest-first. cursor is the created_at of
// the last item already seen (RFC3339); empty starts from the newest.
func (s *Service) List(ctx context.Context, cursor string, limit int) (Page, error) {
	if limit <= 0 || limit > maxPageLen {
		limit = defaultPageLen
	}
	var before *time.Time
	if cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, cursor)
		if err == nil {
			before = &t
		}
	}

	items, err := s.repo.List(ctx, s.pool, before, limit+1)
	if err != nil {
		return Page{}, err
	}

	page := Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1].CreatedAt.Format(time.RFC3339Nano)
		page.NextCursor = &last
		page.HasMore = true
	}
	if page.Items == nil {
		page.Items = []DTO{}
	}
	return page, nil
}

// Delete removes a feedback entry. Only the CEO (clearance level 1) may do so.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, actor Actor) error {
	if actor.RoleLevel != 1 {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, s.pool, id)
}
