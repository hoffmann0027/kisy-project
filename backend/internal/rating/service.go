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

// ChangePublisher notifies every connected client that the shared rating board
// changed, so they refetch. Injected to avoid a rating→ws cycle; may be nil.
type ChangePublisher func()

type Service struct {
	userLevel UserLookup
	pool      *pgxpool.Pool
	repo      Repository
	changed   ChangePublisher
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// SetChangePublisher wires real-time board-change notifications.
func (s *Service) SetChangePublisher(p ChangePublisher) { s.changed = p }

// UserLookup resolves an active user's clearance; false for an unknown or
// inactive one. Injected to avoid a rating→users cycle.
type UserLookup func(ctx context.Context, id uuid.UUID) (int, bool)

// SetUserLookup wires how a would-be member's clearance is found.
func (s *Service) SetUserLookup(f UserLookup) { s.userLevel = f }

func (s *Service) notify() {
	if s.changed != nil {
		s.changed()
	}
}

// Board returns the projects the actor may see (clearance L sees a project when
// L <= min_level), with their tasks grouped underneath.
func (s *Service) Board(ctx context.Context, actorLevel int) (Board, error) {
	projects, err := s.repo.ListProjects(ctx, s.pool, actorLevel)
	if err != nil {
		return Board{}, err
	}
	tasks, err := s.repo.ListTasks(ctx, s.pool, actorLevel)
	if err != nil {
		return Board{}, err
	}
	members, err := s.repo.ListMembers(ctx, s.pool, actorLevel)
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
	for _, m := range members {
		if idx, ok := byproject[m.ProjectID]; ok {
			projects[idx].Members = append(projects[idx].Members, m.Assignee)
		}
	}
	if projects == nil {
		projects = []ProjectDTO{}
	}
	return Board{Projects: projects}, nil
}

// Analytics returns the per-project profit share and monthly totals, scoped to
// projects the actor may see.
func (s *Service) Analytics(ctx context.Context, actorLevel int) (AnalyticsDTO, error) {
	a, err := s.repo.Analytics(ctx, s.pool, actorLevel)
	if err != nil {
		return AnalyticsDTO{}, err
	}
	if a.PerProject == nil {
		a.PerProject = []ProjectProfit{}
	}
	if a.Monthly == nil {
		a.Monthly = []MonthlyProfit{}
	}
	if a.Recent == nil {
		a.Recent = []LedgerEntryDTO{}
	}
	return a, nil
}

// ExportFinance returns the profit ledger the actor may see, for CSV export.
func (s *Service) ExportFinance(ctx context.Context, actorLevel int) ([]FinanceRow, error) {
	return s.repo.ListFinance(ctx, s.pool, actorLevel)
}

// CreateProjectInput is validated by the service. MinLevel (1–10) is the
// project's access level: users of clearance L see it when L <= MinLevel.
type CreateProjectInput struct {
	Title       string
	Description *string
	Difficulty  string
	MinLevel    int
}

// CreateProject adds a backlog project. The CEO and managers (levels 1–4)
// may; a manager cannot put it above their own clearance, or they could not
// see what they made.
func (s *Service) CreateProject(ctx context.Context, in CreateProjectInput, actor Actor) (uuid.UUID, error) {
	if !actor.isManager() {
		return uuid.Nil, ErrForbidden
	}
	if !actor.isCEO() && in.MinLevel < actor.RoleLevel {
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
	if in.MinLevel < 1 || in.MinLevel > 10 {
		return uuid.Nil, ErrValidation
	}
	id, err := s.repo.CreateProject(ctx, s.pool, in.Title, in.Description, in.Difficulty, in.MinLevel, actor.UserID)
	if err == nil {
		s.notify()
	}
	return id, err
}

// SetProjectLevel changes a project's access level (1–10) at any stage:
// whoever runs the project, and a manager not above their own clearance.
// Existing task assignees keep their tasks; only visibility changes.
func (s *Service) SetProjectLevel(ctx context.Context, id uuid.UUID, minLevel int, actor Actor) error {
	if minLevel < 1 || minLevel > 10 {
		return ErrValidation
	}
	if _, err := s.managed(ctx, id, actor); err != nil {
		return err
	}
	if !actor.isCEO() && minLevel < actor.RoleLevel {
		return ErrForbidden
	}
	err := s.repo.SetProjectLevel(ctx, s.pool, id, minLevel)
	if err == nil {
		s.notify()
	}
	return err
}

// managed loads a project and refuses the actor who does not run it. A
// project the actor cannot see answers like one that does not exist.
func (s *Service) managed(ctx context.Context, id uuid.UUID, actor Actor) (ProjectRow, error) {
	p, err := s.repo.GetProject(ctx, s.pool, id)
	if err != nil {
		return ProjectRow{}, err
	}
	if p.MinLevel < actor.RoleLevel {
		return ProjectRow{}, ErrNotFound
	}
	if !actor.canManage(p) {
		return ProjectRow{}, ErrForbidden
	}
	return p, nil
}

// DeleteProject removes a project and its tasks/ledger: whoever runs it.
func (s *Service) DeleteProject(ctx context.Context, id uuid.UUID, actor Actor) error {
	if _, err := s.managed(ctx, id, actor); err != nil {
		return err
	}
	err := s.repo.DeleteProject(ctx, s.pool, id)
	if err == nil {
		s.notify()
	}
	return err
}

// CreateTask adds a task to a project's backlog: whoever runs the project.
func (s *Service) CreateTask(ctx context.Context, projectID uuid.UUID, title string, actor Actor) (uuid.UUID, error) {
	title = strings.TrimSpace(title)
	if n := len([]rune(title)); n < 1 || n > 200 {
		return uuid.Nil, ErrValidation
	}
	if _, err := s.managed(ctx, projectID, actor); err != nil {
		return uuid.Nil, err
	}
	id, err := s.repo.CreateTask(ctx, s.pool, projectID, title)
	if err == nil {
		s.notify()
	}
	return id, err
}

// AssignSelf claims a backlog task for the acting user, moving it to
// "in progress". A user may only assign themselves — the handler passes the
// actor's own id, so there is no way to assign anyone else.
func (s *Service) AssignSelf(ctx context.Context, taskID uuid.UUID, actor Actor) error {
	err := s.repo.Assign(ctx, s.pool, taskID, actor.UserID, actor.RoleLevel)
	if err == nil {
		s.notify()
	}
	return err
}

// SetProgress updates a task's progress (0–100). Only the assignee may do so.
// Reaching 100 marks the task done; if that was the project's last task, the
// project is completed (its tasks removed, card moved to the done column) in
// the same transaction.
func (s *Service) SetProgress(ctx context.Context, taskID uuid.UUID, progress int, actor Actor) error {
	if progress < 0 || progress > 100 {
		return ErrValidation
	}
	task, err := s.repo.GetTask(ctx, s.pool, taskID, actor.RoleLevel)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.SetProgress(ctx, tx, taskID, actor.UserID, progress, actor.RoleLevel); err != nil {
		return err // ErrForbidden if not the assignee
	}
	if progress >= 100 {
		allDone, total, err := s.repo.ProjectTasksAllDone(ctx, tx, task.ProjectID)
		if err != nil {
			return err
		}
		if allDone && total > 0 {
			if err := s.repo.CompleteProject(ctx, tx, task.ProjectID); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.notify()
	return nil
}

// ReturnTask sends an in-progress task back to the backlog. The current
// assignee or the CEO may do this.
func (s *Service) ReturnTask(ctx context.Context, taskID uuid.UUID, actor Actor) error {
	task, err := s.repo.GetTask(ctx, s.pool, taskID, actor.RoleLevel)
	if err != nil {
		return err
	}
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == actor.UserID
	if !isAssignee && !actor.isCEO() {
		return ErrForbidden
	}
	err = s.repo.ReturnTask(ctx, s.pool, taskID, actor.RoleLevel)
	if err == nil {
		s.notify()
	}
	return err
}

// DeleteTask removes a task at any stage: whoever runs its project.
func (s *Service) DeleteTask(ctx context.Context, taskID uuid.UUID, actor Actor) error {
	task, err := s.repo.GetTask(ctx, s.pool, taskID, actor.RoleLevel)
	if err != nil {
		return err
	}
	if _, err := s.managed(ctx, task.ProjectID, actor); err != nil {
		return err
	}
	err = s.repo.DeleteTask(ctx, s.pool, taskID)
	if err == nil {
		s.notify()
	}
	return err
}

// FinanceInput carries a new ledger entry (money in integer minor units — euro
// cents).
type FinanceInput struct {
	IncomeKopecks  int64
	ExpenseKopecks int64
	Note           *string
}

// AddFinance records income/expense against a project's profit ledger:
// whoever runs the project, and its members — the row names its author, so
// who answers for the money is always on record.
func (s *Service) AddFinance(ctx context.Context, projectID uuid.UUID, in FinanceInput, actor Actor) error {
	if in.IncomeKopecks < 0 || in.ExpenseKopecks < 0 || (in.IncomeKopecks == 0 && in.ExpenseKopecks == 0) {
		return ErrValidation
	}
	p, err := s.repo.GetProject(ctx, s.pool, projectID)
	if err != nil {
		return err
	}
	if p.MinLevel < actor.RoleLevel {
		return ErrNotFound
	}
	if !actor.canManage(p) {
		member, err := s.repo.IsMember(ctx, s.pool, projectID, actor.UserID)
		if err != nil {
			return err
		}
		if !member {
			return ErrForbidden
		}
	}
	err = s.repo.AddFinance(ctx, s.pool, projectID, nil, in.IncomeKopecks, in.ExpenseKopecks, in.Note, actor.UserID)
	if err == nil {
		s.notify()
	}
	return err
}

// AddMember puts a user on a project: whoever runs it may, and only someone
// who can see the project — its threshold is not raised by membership.
func (s *Service) AddMember(ctx context.Context, projectID, userID uuid.UUID, actor Actor) error {
	p, err := s.managed(ctx, projectID, actor)
	if err != nil {
		return err
	}
	if s.userLevel == nil {
		return ErrForbidden
	}
	level, ok := s.userLevel(ctx, userID)
	if !ok {
		return ErrNotFound
	}
	if level < 1 || level > p.MinLevel {
		return ErrOutOfReach
	}
	if err := s.repo.AddMember(ctx, s.pool, projectID, userID, actor.UserID); err != nil {
		return err
	}
	s.notify()
	return nil
}

// RemoveMember takes a user off a project: whoever runs it may.
func (s *Service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID, actor Actor) error {
	if _, err := s.managed(ctx, projectID, actor); err != nil {
		return err
	}
	if err := s.repo.RemoveMember(ctx, s.pool, projectID, userID); err != nil {
		return err
	}
	s.notify()
	return nil
}
