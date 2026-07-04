package rating

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Board is the full three-column view: projects (with their tasks embedded).
// The client derives the columns — backlog projects, in-progress tasks and
// done tasks — from this single payload.
type Board struct {
	Projects []ProjectDTO `json:"projects"`
}

type Service struct {
	pool *pgxpool.Pool
	repo Repository
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// Board returns every project with its tasks grouped underneath.
func (s *Service) Board(ctx context.Context) (Board, error) {
	projects, err := s.repo.ListProjects(ctx, s.pool)
	if err != nil {
		return Board{}, err
	}
	tasks, err := s.repo.ListTasks(ctx, s.pool)
	if err != nil {
		return Board{}, err
	}

	byproject := make(map[uuid.UUID]int, len(projects))
	for i := range projects {
		byproject[projects[i].ID] = i
	}
	for _, t := range tasks {
		if idx, ok := byproject[t.ProjectID]; ok {
			projects[idx].Tasks = append(projects[idx].Tasks, t)
		}
	}
	if projects == nil {
		projects = []ProjectDTO{}
	}
	return Board{Projects: projects}, nil
}

// Analytics returns the per-project profit share and monthly totals.
func (s *Service) Analytics(ctx context.Context) (AnalyticsDTO, error) {
	a, err := s.repo.Analytics(ctx, s.pool)
	if err != nil {
		return AnalyticsDTO{}, err
	}
	if a.PerProject == nil {
		a.PerProject = []ProjectProfit{}
	}
	if a.Monthly == nil {
		a.Monthly = []MonthlyProfit{}
	}
	return a, nil
}

// CreateProjectInput is validated by the service.
type CreateProjectInput struct {
	Title       string
	Description *string
	Difficulty  string
}

// CreateProject adds a backlog project. Only the CEO may create projects.
func (s *Service) CreateProject(ctx context.Context, in CreateProjectInput, actor Actor) (uuid.UUID, error) {
	if !actor.isCEO() {
		return uuid.Nil, ErrForbidden
	}
	in.Title = strings.TrimSpace(in.Title)
	if n := len([]rune(in.Title)); n < 1 || n > 128 {
		return uuid.Nil, ErrValidation
	}
	if in.Difficulty == "" {
		in.Difficulty = "medium"
	}
	if !validDifficulty[in.Difficulty] {
		return uuid.Nil, ErrValidation
	}
	return s.repo.CreateProject(ctx, s.pool, in.Title, in.Description, in.Difficulty, actor.UserID)
}

// DeleteProject removes a project and its tasks/ledger. CEO only.
func (s *Service) DeleteProject(ctx context.Context, id uuid.UUID, actor Actor) error {
	if !actor.isCEO() {
		return ErrForbidden
	}
	return s.repo.DeleteProject(ctx, s.pool, id)
}

// CreateTask adds a task to a project's backlog. CEO only.
func (s *Service) CreateTask(ctx context.Context, projectID uuid.UUID, title string, actor Actor) (uuid.UUID, error) {
	if !actor.isCEO() {
		return uuid.Nil, ErrForbidden
	}
	title = strings.TrimSpace(title)
	if n := len([]rune(title)); n < 1 || n > 200 {
		return uuid.Nil, ErrValidation
	}
	exists, err := s.repo.ProjectExists(ctx, s.pool, projectID)
	if err != nil {
		return uuid.Nil, err
	}
	if !exists {
		return uuid.Nil, ErrNotFound
	}
	return s.repo.CreateTask(ctx, s.pool, projectID, title)
}

// AssignSelf claims a backlog task for the acting user, moving it to
// "in progress". A user may only assign themselves — the handler passes the
// actor's own id, so there is no way to assign anyone else.
func (s *Service) AssignSelf(ctx context.Context, taskID uuid.UUID, actor Actor) error {
	return s.repo.Assign(ctx, s.pool, taskID, actor.UserID)
}

// SetProgress updates a task's progress (0–100). Only the assignee may do so;
// reaching 100 moves the task to "done".
func (s *Service) SetProgress(ctx context.Context, taskID uuid.UUID, progress int, actor Actor) error {
	if progress < 0 || progress > 100 {
		return ErrValidation
	}
	return s.repo.SetProgress(ctx, s.pool, taskID, actor.UserID, progress)
}

// FinanceInput carries a new ledger entry (money in integer kopecks).
type FinanceInput struct {
	IncomeKopecks  int64
	ExpenseKopecks int64
	Note           *string
}

// AddFinance records income/expense against a completed task. Only the task's
// assignee or the CEO may record finances, and only once the task is done.
func (s *Service) AddFinance(ctx context.Context, taskID uuid.UUID, in FinanceInput, actor Actor) error {
	if in.IncomeKopecks < 0 || in.ExpenseKopecks < 0 || (in.IncomeKopecks == 0 && in.ExpenseKopecks == 0) {
		return ErrValidation
	}
	task, err := s.repo.GetTask(ctx, s.pool, taskID)
	if err != nil {
		return err
	}
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == actor.UserID
	if !isAssignee && !actor.isCEO() {
		return ErrForbidden
	}
	if task.Status != StatusDone {
		return ErrValidation
	}
	return s.repo.AddFinance(ctx, s.pool, task.ProjectID, &taskID, in.IncomeKopecks, in.ExpenseKopecks, in.Note, actor.UserID)
}
