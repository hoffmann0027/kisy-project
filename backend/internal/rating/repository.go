package rating

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// TaskRow is the minimal task state used for authorization decisions.
type TaskRow struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	AssigneeID *uuid.UUID
	Status     string
	Progress   int
}

// Repository is the persistence port for the rating board.
type Repository interface {
	// The *actorLevel* args scope results to projects the actor may see
	// (their clearance L can see a project when L <= min_level).
	ListProjects(ctx context.Context, q db.DBTX, actorLevel int) ([]ProjectDTO, error)
	ListTasks(ctx context.Context, q db.DBTX, actorLevel int) ([]TaskDTO, error)
	Analytics(ctx context.Context, q db.DBTX, actorLevel int) (AnalyticsDTO, error)

	CreateProject(ctx context.Context, q db.DBTX, title string, description *string, difficulty string, minLevel int, createdBy uuid.UUID) (uuid.UUID, error)
	DeleteProject(ctx context.Context, q db.DBTX, id uuid.UUID) error
	ProjectExists(ctx context.Context, q db.DBTX, id uuid.UUID) (bool, error)
	CreateTask(ctx context.Context, q db.DBTX, projectID uuid.UUID, title string) (uuid.UUID, error)

	GetTask(ctx context.Context, q db.DBTX, id uuid.UUID) (TaskRow, error)
	// ProjectTasksAllDone reports whether a project has at least one task and
	// every task is done — the trigger to complete the project.
	ProjectTasksAllDone(ctx context.Context, q db.DBTX, projectID uuid.UUID) (allDone bool, total int, err error)
	// CompleteProject marks the project done and removes its tasks.
	CompleteProject(ctx context.Context, q db.DBTX, projectID uuid.UUID) error
	// ReturnTask sends a task back to the backlog (unassigned, progress 0).
	ReturnTask(ctx context.Context, q db.DBTX, taskID uuid.UUID) error
	// DeleteTask removes a task outright (CEO override).
	DeleteTask(ctx context.Context, q db.DBTX, taskID uuid.UUID) error
	// Assign claims a backlog task for the user (self-assignment). Returns
	// ErrAlreadyClaimed if the task is not an unassigned backlog task.
	Assign(ctx context.Context, q db.DBTX, taskID, userID uuid.UUID) error
	// SetProgress updates progress for the assignee only, flipping status to
	// done at 100. Returns ErrForbidden if the user is not the assignee.
	SetProgress(ctx context.Context, q db.DBTX, taskID, userID uuid.UUID, progress int) error

	AddFinance(ctx context.Context, q db.DBTX, projectID uuid.UUID, taskID *uuid.UUID, income, expense int64, note *string, createdBy uuid.UUID) error
	// ListFinance returns ledger entries (scoped to accessible projects) joined
	// to project/task/author for CSV export, oldest first.
	ListFinance(ctx context.Context, q db.DBTX, actorLevel int) ([]FinanceRow, error)
}

// FinanceRow is one exported ledger line.
type FinanceRow struct {
	ProjectTitle   string
	TaskTitle      *string
	IncomeKopecks  int64
	ExpenseKopecks int64
	Note           *string
	AuthorName     string
	CreatedAt      time.Time
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) ListProjects(ctx context.Context, q db.DBTX, actorLevel int) ([]ProjectDTO, error) {
	rows, err := q.Query(ctx, `
		SELECT p.id, p.title, p.description, p.difficulty, p.min_level, p.status, p.created_by, p.created_at,
		       COALESCE(fp.profit, 0)
		FROM rating_projects p
		LEFT JOIN (
			SELECT project_id, SUM(income_kopecks - expense_kopecks) AS profit
			FROM rating_finance_entries GROUP BY project_id
		) fp ON fp.project_id = p.id
		WHERE p.min_level >= $1
		ORDER BY p.created_at`, actorLevel)
	if err != nil {
		return nil, fmt.Errorf("rating: list projects: %w", err)
	}
	defer rows.Close()

	var out []ProjectDTO
	for rows.Next() {
		var p ProjectDTO
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty, &p.MinLevel, &p.Status, &p.CreatedBy, &p.CreatedAt, &p.TotalProfitKopecks); err != nil {
			return nil, fmt.Errorf("rating: scan project: %w", err)
		}
		p.Tasks = []TaskDTO{}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListTasks(ctx context.Context, q db.DBTX, actorLevel int) ([]TaskDTO, error) {
	rows, err := q.Query(ctx, `
		SELECT t.id, t.project_id, p.title, t.title, t.assignee_id, u.display_name, u.avatar_url,
		       t.progress, t.status, t.created_at, COALESCE(ft.profit, 0)
		FROM rating_tasks t
		JOIN rating_projects p ON p.id = t.project_id
		LEFT JOIN users u ON u.id = t.assignee_id
		LEFT JOIN (
			SELECT task_id, SUM(income_kopecks - expense_kopecks) AS profit
			FROM rating_finance_entries WHERE task_id IS NOT NULL GROUP BY task_id
		) ft ON ft.task_id = t.id
		WHERE p.min_level >= $1
		ORDER BY t.created_at`, actorLevel)
	if err != nil {
		return nil, fmt.Errorf("rating: list tasks: %w", err)
	}
	defer rows.Close()

	var out []TaskDTO
	for rows.Next() {
		var t TaskDTO
		var assigneeID *uuid.UUID
		var displayName *string
		var avatarURL *string
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ProjectTitle, &t.Title, &assigneeID, &displayName, &avatarURL,
			&t.Progress, &t.Status, &t.CreatedAt, &t.TotalProfitKopecks); err != nil {
			return nil, fmt.Errorf("rating: scan task: %w", err)
		}
		if assigneeID != nil {
			name := ""
			if displayName != nil {
				name = *displayName
			}
			t.Assignee = &Assignee{ID: *assigneeID, DisplayName: name, AvatarURL: avatarURL}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Analytics(ctx context.Context, q db.DBTX, actorLevel int) (AnalyticsDTO, error) {
	var a AnalyticsDTO

	perRows, err := q.Query(ctx, `
		SELECT p.id, p.title, COALESCE(SUM(f.income_kopecks - f.expense_kopecks), 0)
		FROM rating_projects p
		LEFT JOIN rating_finance_entries f ON f.project_id = p.id
		WHERE p.min_level >= $1
		GROUP BY p.id, p.title
		ORDER BY 3 DESC`, actorLevel)
	if err != nil {
		return a, fmt.Errorf("rating: analytics per-project: %w", err)
	}
	defer perRows.Close()
	for perRows.Next() {
		var pp ProjectProfit
		if err := perRows.Scan(&pp.ProjectID, &pp.Title, &pp.ProfitKopecks); err != nil {
			return a, fmt.Errorf("rating: scan per-project: %w", err)
		}
		a.PerProject = append(a.PerProject, pp)
	}
	if err := perRows.Err(); err != nil {
		return a, err
	}

	monRows, err := q.Query(ctx, `
		SELECT to_char(date_trunc('month', f.created_at), 'YYYY-MM'),
		       SUM(f.income_kopecks - f.expense_kopecks)
		FROM rating_finance_entries f
		JOIN rating_projects p ON p.id = f.project_id
		WHERE p.min_level >= $1
		GROUP BY 1 ORDER BY 1`, actorLevel)
	if err != nil {
		return a, fmt.Errorf("rating: analytics monthly: %w", err)
	}
	defer monRows.Close()
	for monRows.Next() {
		var mp MonthlyProfit
		if err := monRows.Scan(&mp.Month, &mp.ProfitKopecks); err != nil {
			return a, fmt.Errorf("rating: scan monthly: %w", err)
		}
		a.Monthly = append(a.Monthly, mp)
	}
	return a, monRows.Err()
}

func (r *PostgresRepository) CreateProject(ctx context.Context, q db.DBTX, title string, description *string, difficulty string, minLevel int, createdBy uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx, `
		INSERT INTO rating_projects (title, description, difficulty, min_level, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		title, description, difficulty, minLevel, createdBy).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("rating: create project: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) DeleteProject(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM rating_projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("rating: delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ProjectExists(ctx context.Context, q db.DBTX, id uuid.UUID) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM rating_projects WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("rating: project exists: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CreateTask(ctx context.Context, q db.DBTX, projectID uuid.UUID, title string) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx, `
		INSERT INTO rating_tasks (project_id, title) VALUES ($1, $2) RETURNING id`,
		projectID, title).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("rating: create task: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) GetTask(ctx context.Context, q db.DBTX, id uuid.UUID) (TaskRow, error) {
	var t TaskRow
	err := q.QueryRow(ctx, `SELECT id, project_id, assignee_id, status, progress FROM rating_tasks WHERE id = $1`, id).
		Scan(&t.ID, &t.ProjectID, &t.AssigneeID, &t.Status, &t.Progress)
	if errors.Is(err, pgx.ErrNoRows) {
		return TaskRow{}, ErrNotFound
	}
	if err != nil {
		return TaskRow{}, fmt.Errorf("rating: get task: %w", err)
	}
	return t, nil
}

func (r *PostgresRepository) Assign(ctx context.Context, q db.DBTX, taskID, userID uuid.UUID) error {
	tag, err := q.Exec(ctx, `
		UPDATE rating_tasks
		SET assignee_id = $2, status = 'in_progress', updated_at = now()
		WHERE id = $1 AND assignee_id IS NULL AND status = 'backlog'`,
		taskID, userID)
	if err != nil {
		return fmt.Errorf("rating: assign: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadyClaimed
	}
	return nil
}

func (r *PostgresRepository) ProjectTasksAllDone(ctx context.Context, q db.DBTX, projectID uuid.UUID) (bool, int, error) {
	var total, done int
	err := q.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE status = 'done')
		FROM rating_tasks WHERE project_id = $1`, projectID).Scan(&total, &done)
	if err != nil {
		return false, 0, fmt.Errorf("rating: count tasks: %w", err)
	}
	return total > 0 && total == done, total, nil
}

func (r *PostgresRepository) CompleteProject(ctx context.Context, q db.DBTX, projectID uuid.UUID) error {
	if _, err := q.Exec(ctx, `DELETE FROM rating_tasks WHERE project_id = $1`, projectID); err != nil {
		return fmt.Errorf("rating: delete tasks: %w", err)
	}
	if _, err := q.Exec(ctx, `UPDATE rating_projects SET status = 'done', completed_at = now() WHERE id = $1`, projectID); err != nil {
		return fmt.Errorf("rating: complete project: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ReturnTask(ctx context.Context, q db.DBTX, taskID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		UPDATE rating_tasks SET assignee_id = NULL, progress = 0, status = 'backlog', updated_at = now()
		WHERE id = $1`, taskID)
	if err != nil {
		return fmt.Errorf("rating: return task: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteTask(ctx context.Context, q db.DBTX, taskID uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM rating_tasks WHERE id = $1`, taskID)
	if err != nil {
		return fmt.Errorf("rating: delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) SetProgress(ctx context.Context, q db.DBTX, taskID, userID uuid.UUID, progress int) error {
	tag, err := q.Exec(ctx, `
		UPDATE rating_tasks
		SET progress = $3,
		    status = CASE WHEN $3 >= 100 THEN 'done' ELSE 'in_progress' END,
		    updated_at = now()
		WHERE id = $1 AND assignee_id = $2 AND status <> 'backlog'`,
		taskID, userID, progress)
	if err != nil {
		return fmt.Errorf("rating: set progress: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

func (r *PostgresRepository) ListFinance(ctx context.Context, q db.DBTX, actorLevel int) ([]FinanceRow, error) {
	rows, err := q.Query(ctx, `
		SELECT p.title, t.title, f.income_kopecks, f.expense_kopecks, f.note, u.display_name, f.created_at
		FROM rating_finance_entries f
		JOIN rating_projects p ON p.id = f.project_id
		LEFT JOIN rating_tasks t ON t.id = f.task_id
		JOIN users u ON u.id = f.created_by
		WHERE p.min_level >= $1
		ORDER BY f.created_at`, actorLevel)
	if err != nil {
		return nil, fmt.Errorf("rating: list finance: %w", err)
	}
	defer rows.Close()

	var out []FinanceRow
	for rows.Next() {
		var fr FinanceRow
		if err := rows.Scan(&fr.ProjectTitle, &fr.TaskTitle, &fr.IncomeKopecks, &fr.ExpenseKopecks, &fr.Note, &fr.AuthorName, &fr.CreatedAt); err != nil {
			return nil, fmt.Errorf("rating: scan finance: %w", err)
		}
		out = append(out, fr)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) AddFinance(ctx context.Context, q db.DBTX, projectID uuid.UUID, taskID *uuid.UUID, income, expense int64, note *string, createdBy uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO rating_finance_entries (project_id, task_id, income_kopecks, expense_kopecks, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		projectID, taskID, income, expense, note, createdBy)
	if err != nil {
		return fmt.Errorf("rating: add finance: %w", err)
	}
	return nil
}
