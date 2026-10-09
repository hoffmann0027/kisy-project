// Package rating implements the "Рейтинг" project board: a three-column
// kanban (projects → in progress → done) with a per-project net-profit ledger
// and analytics. All authority (CEO creates projects/tasks; a user may assign
// only themselves; only the assignee moves progress; assignee or CEO record
// finances) is enforced in the service, never merely hidden in the UI.
package rating

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("rating: not found")
	ErrForbidden      = errors.New("rating: not permitted")
	ErrValidation     = errors.New("rating: invalid input")
	ErrAlreadyClaimed = errors.New("rating: task already has an assignee")
	// ErrOutOfReach refuses a member who could not see the project: its
	// threshold is above their clearance.
	ErrOutOfReach = errors.New("rating: the user cannot see this project")
)

// ManagerMaxLevel is the weakest clearance that runs projects of its own:
// levels 1–4 create projects, and in those they created they add tasks and
// members and record money. The CEO does all of that everywhere.
const ManagerMaxLevel = 4

// Task lifecycle columns.
const (
	StatusBacklog    = "backlog"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
)

// Project lifecycle. A project is "active" until all its tasks complete, then
// "done" (its tasks are removed and the card moves to the done column).
const (
	ProjectActive = "active"
	ProjectDone   = "done"
)

var validDifficulty = map[string]bool{"easy": true, "medium": true, "hard": true}

// Actor identifies the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

func (a Actor) isCEO() bool { return a.RoleLevel == 1 }

// isManager reports whether the actor may create projects of their own.
func (a Actor) isManager() bool { return a.RoleLevel >= 1 && a.RoleLevel <= ManagerMaxLevel }

// ProjectRow is the project state authorization is decided on.
type ProjectRow struct {
	ID        uuid.UUID
	CreatedBy uuid.UUID
	MinLevel  int
	Status    string
}

// canManage reports whether the actor runs the project: the CEO, or a
// manager who created it.
func (a Actor) canManage(p ProjectRow) bool {
	return a.isCEO() || (a.isManager() && p.CreatedBy == a.UserID)
}

// Assignee is the public identity of a task's executor.
type Assignee struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
}

// TaskDTO is one task card. TotalProfitKopecks is the sum of the task's
// ledger entries (income − expense), used to sort the "done" column.
type TaskDTO struct {
	ID                 uuid.UUID `json:"id"`
	ProjectID          uuid.UUID `json:"projectId"`
	ProjectTitle       string    `json:"projectTitle"`
	Title              string    `json:"title"`
	Assignee           *Assignee `json:"assignee"`
	Progress           int       `json:"progress"`
	Status             string    `json:"status"`
	TotalProfitKopecks int64     `json:"totalProfitKopecks"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ProjectDTO is a project with its tasks embedded. The money is the sum of
// its ledger: income and expense apart, and their difference as profit.
type ProjectDTO struct {
	ID                  uuid.UUID `json:"id"`
	Title               string    `json:"title"`
	Description         *string   `json:"description"`
	Difficulty          string    `json:"difficulty"`
	MinLevel            int       `json:"minLevel"`
	Status              string    `json:"status"`
	CreatedBy           uuid.UUID `json:"createdBy"`
	TotalIncomeKopecks  int64     `json:"totalIncomeKopecks"`
	TotalExpenseKopecks int64     `json:"totalExpenseKopecks"`
	TotalProfitKopecks  int64     `json:"totalProfitKopecks"`
	Tasks               []TaskDTO `json:"tasks"`
	// Members answer for the project: they record its money. Who may change
	// the list is the project's creator (a manager) or the CEO.
	Members   []Assignee `json:"members"`
	CreatedAt time.Time  `json:"createdAt"`
	// UpdatedAt is when anything last happened to the project: a task taken,
	// moved or returned, money recorded, the project completed.
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

// FinanceEntryDTO is one ledger record.
type FinanceEntryDTO struct {
	ID             uuid.UUID `json:"id"`
	IncomeKopecks  int64     `json:"incomeKopecks"`
	ExpenseKopecks int64     `json:"expenseKopecks"`
	ProfitKopecks  int64     `json:"profitKopecks"`
	Note           *string   `json:"note"`
	CreatedBy      uuid.UUID `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AnalyticsDTO powers the dashboard's charts: each project's share of the
// money, the monthly income/expense/profit series, and the latest ledger
// entries as a feed of what happened.
type AnalyticsDTO struct {
	PerProject []ProjectProfit  `json:"perProject"`
	Monthly    []MonthlyProfit  `json:"monthly"`
	Recent     []LedgerEntryDTO `json:"recent"`
}

type ProjectProfit struct {
	ProjectID      uuid.UUID `json:"projectId"`
	Title          string    `json:"title"`
	IncomeKopecks  int64     `json:"incomeKopecks"`
	ExpenseKopecks int64     `json:"expenseKopecks"`
	ProfitKopecks  int64     `json:"profitKopecks"`
}

type MonthlyProfit struct {
	Month          string `json:"month"` // "YYYY-MM"
	IncomeKopecks  int64  `json:"incomeKopecks"`
	ExpenseKopecks int64  `json:"expenseKopecks"`
	ProfitKopecks  int64  `json:"profitKopecks"`
}

// LedgerEntryDTO is one recorded income or expense, with the project it was
// recorded against and who recorded it.
type LedgerEntryDTO struct {
	ID             uuid.UUID `json:"id"`
	ProjectID      uuid.UUID `json:"projectId"`
	ProjectTitle   string    `json:"projectTitle"`
	IncomeKopecks  int64     `json:"incomeKopecks"`
	ExpenseKopecks int64     `json:"expenseKopecks"`
	Note           *string   `json:"note"`
	AuthorName     string    `json:"authorName"`
	CreatedAt      time.Time `json:"createdAt"`
}

// RecentLedgerLimit is how many of the latest ledger entries the analytics
// carry: a feed, not the whole history (that is the CSV export).
const RecentLedgerLimit = 12
