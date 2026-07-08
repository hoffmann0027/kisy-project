// Package conditions owns the promotion-requirements ladder ("Условия
// повышения уровня"). The CEO (clearance level 1) writes and edits the rule
// for every target rank 1..9. Any other user may read only the single rule for
// their own next level (their clearance minus one), so they never see the rest
// of the ladder in advance. That visibility rule is enforced here in the
// service, never merely hidden in the UI.
package conditions

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

// MaxBodyLen caps a single condition's text.
const MaxBodyLen = 4000

var (
	ErrForbidden  = errors.New("conditions: not permitted")
	ErrValidation = errors.New("conditions: invalid input")
	ErrNoNext     = errors.New("conditions: no next level")
)

// DTO is one promotion rule: the requirement to reach targetLevel.
type DTO struct {
	TargetLevel int       `json:"targetLevel"`
	Body        string    `json:"body"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Actor identifies the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

func (a Actor) isCEO() bool { return a.RoleLevel == 1 }

// Repository is the persistence port.
type Repository interface {
	List(ctx context.Context, q db.DBTX) ([]DTO, error)
	Get(ctx context.Context, q db.DBTX, level int) (DTO, error)
	Upsert(ctx context.Context, q db.DBTX, level int, body string, updatedBy uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) List(ctx context.Context, q db.DBTX) ([]DTO, error) {
	rows, err := q.Query(ctx, `SELECT target_level, body, updated_at FROM level_conditions ORDER BY target_level`)
	if err != nil {
		return nil, fmt.Errorf("conditions: list: %w", err)
	}
	defer rows.Close()
	var out []DTO
	for rows.Next() {
		var d DTO
		if err := rows.Scan(&d.TargetLevel, &d.Body, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("conditions: scan: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Get(ctx context.Context, q db.DBTX, level int) (DTO, error) {
	var d DTO
	err := q.QueryRow(ctx, `SELECT target_level, body, updated_at FROM level_conditions WHERE target_level = $1`, level).
		Scan(&d.TargetLevel, &d.Body, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DTO{}, ErrNoNext
	}
	if err != nil {
		return DTO{}, fmt.Errorf("conditions: get: %w", err)
	}
	return d, nil
}

func (r *PostgresRepository) Upsert(ctx context.Context, q db.DBTX, level int, body string, updatedBy uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO level_conditions (target_level, body, updated_at, updated_by)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (target_level)
		DO UPDATE SET body = EXCLUDED.body, updated_at = now(), updated_by = EXCLUDED.updated_by`,
		level, body, updatedBy)
	if err != nil {
		return fmt.Errorf("conditions: upsert: %w", err)
	}
	return nil
}

// Service holds the promotion-ladder business logic.
type Service struct {
	pool *pgxpool.Pool
	repo Repository
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// List returns all nine rules. CEO only.
func (s *Service) List(ctx context.Context, actor Actor) ([]DTO, error) {
	if !actor.isCEO() {
		return nil, ErrForbidden
	}
	items, err := s.repo.List(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []DTO{}
	}
	return items, nil
}

// Next returns the single rule for the actor's next level (their clearance
// minus one). The CEO is already at the top and has no next level.
func (s *Service) Next(ctx context.Context, actor Actor) (DTO, error) {
	target := actor.RoleLevel - 1
	if target < 1 {
		return DTO{}, ErrNoNext
	}
	return s.repo.Get(ctx, s.pool, target)
}

// Set writes the rule for a target level. CEO only.
func (s *Service) Set(ctx context.Context, level int, body string, actor Actor) error {
	if !actor.isCEO() {
		return ErrForbidden
	}
	if level < 1 || level > 9 {
		return ErrValidation
	}
	body = strings.TrimSpace(body)
	if len([]rune(body)) > MaxBodyLen {
		return ErrValidation
	}
	return s.repo.Upsert(ctx, s.pool, level, body, actor.UserID)
}
